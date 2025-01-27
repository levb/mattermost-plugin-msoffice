// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugin

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	pluginapiclient "github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/api"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/command"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/config"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/engine"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/jobs"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/telemetry"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/tracker"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/bot"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/flow"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/httputils"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/oauth2connect"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/pluginapi"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/utils/settingspanel"
)

type Env struct {
	engine.Env
	bot                   bot.Bot
	jobManager            *jobs.JobManager
	notificationProcessor engine.NotificationProcessor
	httpHandler           *httputils.Handler
	configError           error
}

type Plugin struct {
	plugin.MattermostPlugin

	envLock         *sync.RWMutex
	env             Env
	Templates       map[string]*template.Template
	telemetryClient telemetry.Client
}

func NewWithEnv(env engine.Env) *Plugin {
	return &Plugin{
		env: Env{
			Env: env,
		},
		envLock: &sync.RWMutex{},
	}
}

func (p *Plugin) OnActivate() error {
	pluginAPIClient := pluginapiclient.NewClient(p.API, p.Driver)
	stored := config.StoredConfig{}
	err := p.API.LoadPluginConfiguration(&stored)
	if err != nil {
		return errors.WithMessage(err, "failed to load plugin configuration")
	}

	mattermostSiteURL := pluginAPIClient.Configuration.GetConfig().ServiceSettings.SiteURL
	if mattermostSiteURL == nil {
		return errors.New("please configure the Mattermost server's SiteURL, then restart the plugin")
	}

	if errConfig := p.env.Remote.CheckConfiguration(stored); errConfig != nil {
		return errors.Wrap(errConfig, "failed to configure")
	}

	p.initEnv(&p.env, "")
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return errors.Wrap(err, "couldn't get bundle path")
	}
	err = p.loadTemplates(bundlePath)
	if err != nil {
		return err
	}

	err = command.Register(pluginAPIClient)
	if err != nil {
		return errors.Wrap(err, "failed to register command")
	}

	// Telemetry client
	p.telemetryClient, err = telemetry.NewRudderClient()
	if err != nil {
		p.API.LogWarn("Telemetry client not started", "error", err.Error())
	}

	// Get config values
	p.updateEnv(func(e *Env) {
		e.Dependencies.Tracker = tracker.New(
			telemetry.NewTracker(
				p.telemetryClient,
				p.API.GetDiagnosticId(),
				p.API.GetServerVersion(),
				e.PluginID,
				e.PluginVersion,
				config.Provider.TelemetryShortName,
				telemetry.NewTrackerConfig(p.API.GetConfig()),
				telemetry.NewLogger(p.API),
			),
		)
		e.bot = e.bot.WithConfig(stored.Config)
		e.Dependencies.Store = store.NewPluginStore(p.API, e.bot, e.bot, e.Dependencies.Tracker, e.Provider.Features.EncryptedStore, []byte(e.EncryptionKey))
	})

	return nil
}

func (p *Plugin) OnDeactivate() error {
	if p.telemetryClient != nil {
		err := p.telemetryClient.Close()
		if err != nil {
			p.env.Logger.Warnf("OnDeactivate: Failed to close telemetryClient. err=%v", err)
		}
	}

	e := p.getEnv()
	if e.jobManager != nil {
		if err := e.jobManager.Close(); err != nil {
			p.env.Logger.Warnf("OnDeactivate: Failed to close job manager. err=%v", err)
			return err
		}
	}
	return nil
}

func (p *Plugin) OnConfigurationChange() (err error) {
	defer func() {
		p.updateEnv(func(e *Env) {
			e.configError = err
		})
	}()

	env := p.getEnv()
	stored := config.StoredConfig{}
	err = p.API.LoadPluginConfiguration(&stored)
	if err != nil {
		return errors.WithMessage(err, "failed to load plugin configuration")
	}

	mattermostSiteURL := p.API.GetConfig().ServiceSettings.SiteURL
	if mattermostSiteURL == nil {
		return errors.New("plugin requires Mattermost Site URL to be set")
	}
	mattermostURL, err := url.Parse(*mattermostSiteURL)
	if err != nil {
		return err
	}
	pluginURLPath := "/plugins/" + url.PathEscape(env.Config.PluginID)
	pluginURL := strings.TrimRight(*mattermostSiteURL, "/") + pluginURLPath

	p.updateEnv(func(e *Env) {
		p.initEnv(e, pluginURL)

		e.StoredConfig = stored
		e.Config.MattermostSiteURL = *mattermostSiteURL
		e.Config.MattermostSiteHostname = mattermostURL.Hostname()
		e.Config.PluginURL = pluginURL
		e.Config.PluginURLPath = pluginURLPath

		e.bot = e.bot.WithConfig(stored.Config)
		e.Dependencies.Remote = remote.Makers[config.Provider.Name](e.Config, e.bot)

		mscalendarBot := engine.NewMSCalendarBot(e.bot, e.Env, pluginURL)

		e.Dependencies.Logger = e.bot

		// reload tracker behavior looking to some key config changes
		if e.Dependencies.Tracker != nil {
			e.Dependencies.Tracker.ReloadConfig(p.API.GetConfig())
		} else {
			e.Dependencies.Tracker = tracker.New(
				telemetry.NewTracker(
					p.telemetryClient,
					p.API.GetDiagnosticId(),
					p.API.GetServerVersion(),
					e.PluginID,
					e.PluginVersion,
					config.Provider.TelemetryShortName,
					telemetry.NewTrackerConfig(p.API.GetConfig()),
					telemetry.NewLogger(p.API),
				),
			)
		}

		e.Dependencies.Poster = e.bot
		e.Dependencies.Welcomer = mscalendarBot
		e.Dependencies.Store = store.NewPluginStore(p.API, e.bot, e.bot, e.Dependencies.Tracker, e.Provider.Features.EncryptedStore, []byte(e.EncryptionKey))
		e.Dependencies.SettingsPanel = engine.NewSettingsPanel(
			e.bot,
			e.Dependencies.Store,
			e.Dependencies.Store,
			"/settings",
			pluginURL,
			func(userID string) engine.Engine {
				return engine.New(e.Env, userID)
			},
			e.Provider.Features,
		)

		welcomeFlow := engine.NewWelcomeFlow(e.bot, e.Dependencies.Welcomer, e.Provider.Features)
		e.bot.RegisterFlow(welcomeFlow, mscalendarBot)

		if e.Provider.Features.EventNotifications {
			if e.notificationProcessor == nil {
				e.notificationProcessor = engine.NewNotificationProcessor(e.Env)
			} else {
				e.notificationProcessor.Configure(e.Env)
			}
		}

		e.httpHandler = httputils.NewHandler()
		oauth2connect.Init(e.httpHandler, engine.NewOAuth2App(e.Env), config.Provider)
		flow.Init(e.httpHandler, welcomeFlow, mscalendarBot)
		settingspanel.Init(e.httpHandler, e.Dependencies.SettingsPanel)
		api.Init(e.httpHandler, e.Env, e.notificationProcessor)

		if e.jobManager == nil {
			e.jobManager = jobs.NewJobManager(p.API, e.Env)
			e.jobManager.AddJob(jobs.NewStatusSyncJob())
			e.jobManager.AddJob(jobs.NewDailySummaryJob())
			e.jobManager.AddJob(jobs.NewRenewJob())
		}
	})

	return nil
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	env := p.getEnv()
	if env.configError != nil {
		p.API.LogError("Error occurred while getting env", "err", env.configError.Error())
		return nil, model.NewAppError("mscalendarplugin.ExecuteCommand", "Unable to execute command.", nil, env.configError.Error(), http.StatusInternalServerError)
	}

	cmd := command.Command{
		Context:   c,
		Args:      args,
		ChannelID: args.ChannelId,
		Config:    env.Config,
		Engine:    engine.New(env.Env, args.UserId),
	}
	out, mustRedirectToDM, err := cmd.Handle()
	if err != nil {
		p.API.LogError("Error occurred while running the command", "args", args, "err", err.Error())
		return nil, model.NewAppError("mscalendarplugin.ExecuteCommand", "Unable to execute command.", nil, err.Error(), http.StatusInternalServerError)
	}

	if out != "" {
		env.Poster.Ephemeral(args.UserId, args.ChannelId, out)
	}

	response := &model.CommandResponse{}
	if mustRedirectToDM {
		t, appErr := p.API.GetTeam(args.TeamId)
		if appErr != nil {
			return nil, model.NewAppError("mscalendarplugin.ExecuteCommand", "Unable to execute command.", nil, appErr.Error(), http.StatusInternalServerError)
		}
		dmURL := fmt.Sprintf("%s/%s/messages/@%s",
			env.MattermostSiteURL,
			url.PathEscape(t.Name),
			url.PathEscape(config.Provider.BotUsername))
		response.GotoLocation = dmURL
	}

	return response, nil
}

func (p *Plugin) ServeHTTP(_ *plugin.Context, w http.ResponseWriter, req *http.Request) {
	env := p.getEnv()
	if env.configError != nil {
		p.API.LogError("Error occurred while getting env", "err", env.configError.Error())
		http.Error(w, env.configError.Error(), http.StatusInternalServerError)
		return
	}

	env.httpHandler.ServeHTTP(w, req)
}

func (p *Plugin) getEnv() Env {
	p.envLock.RLock()
	defer p.envLock.RUnlock()
	return p.env
}

func (p *Plugin) updateEnv(f func(*Env)) Env {
	p.envLock.Lock()
	defer p.envLock.Unlock()

	f(&p.env)
	return p.env
}

func (p *Plugin) loadTemplates(bundlePath string) error {
	if p.Templates != nil {
		return nil
	}

	templatesPath := filepath.Join(bundlePath, "assets", "templates")
	templates := make(map[string]*template.Template)
	err := filepath.Walk(templatesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		tmpl, err := template.ParseFiles(path)
		if err != nil {
			return nil
		}
		key := path[len(templatesPath):]
		templates[key] = tmpl
		return nil
	})
	if err != nil {
		return errors.WithMessage(err, "OnActivate/loadTemplates failed")
	}
	p.Templates = templates
	return nil
}

func (p *Plugin) initEnv(e *Env, pluginURL string) error {
	e.Dependencies.PluginAPI = pluginapi.New(p.API)

	if e.bot == nil {
		e.bot = bot.New(p.API, pluginURL)
		err := e.bot.Ensure(
			&model.Bot{
				Username:    e.Provider.BotUsername,
				DisplayName: e.Provider.BotDisplayName,
				Description: fmt.Sprintf(config.BotDescription, e.Provider.DisplayName),
			},
			filepath.Join("assets", fmt.Sprintf("profile-%s.png", e.Provider.Name)),
		)
		if err != nil {
			return errors.Wrap(err, "failed to ensure bot account")
		}
	}

	return nil
}
