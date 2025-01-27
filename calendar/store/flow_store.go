// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package store

import "fmt"

const (
	UpdateStatusPropertyName              = "update_status"
	GetConfirmationPropertyName           = "get_confirmation"
	ReceiveNotificationsDuringMeetingName = "receive_notifications_during_meetings"
	SetCustomStatusPropertyName           = "set_custom_status"
	SubscribePropertyName                 = "subscribe"
	ReceiveUpcomingEventReminderName      = "receive_reminder"
)

func (s *pluginStore) SetProperty(userID, propertyName string, value interface{}) error {
	user, err := s.LoadUser(userID)
	if err != nil {
		return err
	}

	boolValue, _ := value.(bool)
	switch propertyName {
	case UpdateStatusPropertyName:
		stringValue, _ := value.(string)
		user.Settings.UpdateStatusFromOptions = stringValue
		s.Tracker.TrackAutomaticStatusUpdate(userID, stringValue, "flow")
	case GetConfirmationPropertyName:
		user.Settings.GetConfirmation = boolValue
	case SetCustomStatusPropertyName:
		user.Settings.SetCustomStatus = boolValue
	case ReceiveUpcomingEventReminderName:
		user.Settings.ReceiveReminders = boolValue
	default:
		return fmt.Errorf("property %s not found", propertyName)
	}

	err = s.StoreUser(user)
	if err != nil {
		return err
	}

	return nil
}

func (s *pluginStore) SetPostID(userID, propertyName, postID string) error {
	user, err := s.LoadUser(userID)
	if err != nil {
		return err
	}

	if user.WelcomeFlowStatus.PostIDs == nil {
		user.WelcomeFlowStatus.PostIDs = make(map[string]string)
	}

	user.WelcomeFlowStatus.PostIDs[propertyName] = postID

	err = s.StoreUser(user)
	if err != nil {
		return err
	}

	return nil
}

func (s *pluginStore) GetPostID(userID, propertyName string) (string, error) {
	user, err := s.LoadUser(userID)
	if err != nil {
		return "", err
	}

	return user.WelcomeFlowStatus.PostIDs[propertyName], nil
}

func (s *pluginStore) RemovePostID(userID, propertyName string) error {
	user, err := s.LoadUser(userID)
	if err != nil {
		return err
	}

	delete(user.WelcomeFlowStatus.PostIDs, propertyName)

	err = s.StoreUser(user)
	if err != nil {
		return err
	}

	return nil
}

func (s *pluginStore) GetCurrentStep(userID string) (int, error) {
	user, err := s.LoadUser(userID)
	if err != nil {
		return 0, err
	}

	return user.WelcomeFlowStatus.Step, nil
}
func (s *pluginStore) SetCurrentStep(userID string, step int) error {
	user, err := s.LoadUser(userID)
	if err != nil {
		return err
	}

	user.WelcomeFlowStatus.Step = step

	err = s.StoreUser(user)
	if err != nil {
		return err
	}

	return nil
}

func (s *pluginStore) DeleteCurrentStep(userID string) error {
	user, err := s.LoadUser(userID)
	if err != nil {
		return err
	}

	user.WelcomeFlowStatus.Step = 0

	err = s.StoreUser(user)
	if err != nil {
		return err
	}

	return nil
}
