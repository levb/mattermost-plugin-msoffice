// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/remote"
	"github.com/mattermost/mattermost-plugin-mscalendar/calendar/store"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestCreateMyEventSubscription(t *testing.T) {
	mscalendar, mockStore, _, _, _, mockClient, _ := GetMockSetup(t)
	expectedSub := GetMockSubscription()

	tests := []struct {
		name      string
		setupMock func()
		assertion func(sub *store.Subscription, err error)
	}{
		{
			name: "error filtering with the user",
			setupMock: func() {
				mscalendar.client = nil
				mscalendar.actingUser = GetMockUser(nil, nil, MockActingUserID, nil)
				mockStore.EXPECT().LoadUser(MockActingUserID).Return(nil, errors.New("error filtering the user")).Times(1)
			},
			assertion: func(sub *store.Subscription, err error) {
				require.ErrorContains(t, err, "error filtering the user")
			},
		},
		{
			name: "error creating the subscription",
			setupMock: func() {
				mscalendar.client = mockClient
				mscalendar.actingUser = GetMockUser(model.NewString(MockActingUserRemoteID), nil, MockActingUserID, nil)
				mockClient.EXPECT().CreateMySubscription(gomock.Any(), MockActingUserRemoteID).Return(nil, errors.New("error creating the subscription"))
			},
			assertion: func(sub *store.Subscription, err error) {
				require.EqualError(t, err, "error creating the subscription")
			},
		},
		{
			name: "subscription created successfully",
			setupMock: func() {
				mscalendar.client = mockClient
				mscalendar.actingUser = GetMockUser(model.NewString(MockActingUserRemoteID), nil, MockActingUserID, nil)
				mockClient.EXPECT().CreateMySubscription(gomock.Any(), MockActingUserRemoteID).Return(&remote.Subscription{}, nil)
				mockStore.EXPECT().StoreUserSubscription(mscalendar.actingUser.User, expectedSub)
			},
			assertion: func(sub *store.Subscription, err error) {
				require.NoError(t, err)
				require.Equal(t, expectedSub, sub)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			sub, err := mscalendar.CreateMyEventSubscription()

			tt.assertion(sub, err)
		})
	}
}

func TestLoadMyEventSubscription(t *testing.T) {
	mscalendar, mockStore, _, _, mockPluginAPI, mockClient, _ := GetMockSetup(t)
	expectedSubscription := GetMockSubscription()

	tests := []struct {
		name      string
		setupMock func()
		assertion func(sub *store.Subscription, err error)
	}{
		{
			name: "error filtering with user",
			setupMock: func() {
				mscalendar.client = nil
				mscalendar.actingUser = GetMockUser(nil, nil, MockActingUserID, nil)
				mockStore.EXPECT().LoadUser(MockActingUserID).Return(nil, errors.New("error filtering user")).Times(1)
			},
			assertion: func(sub *store.Subscription, err error) {
				require.ErrorContains(t, err, "error filtering user")
			},
		},
		{
			name: "error loading the subscription",
			setupMock: func() {
				mscalendar.client = mockClient
				mscalendar.actingUser = GetMockUser(model.NewString(MockActingUserRemoteID), nil, MockActingUserID, nil)
				mscalendar.actingUser.Settings.EventSubscriptionID = MockEventSubscriptionID
				mockPluginAPI.EXPECT().GetMattermostUser(MockActingUserID).Return(&model.User{}, nil)
				mockStore.EXPECT().LoadSubscription(MockEventSubscriptionID).Return(nil, errors.New("error loading the subscription")).Times(1)
			},
			assertion: func(sub *store.Subscription, err error) {
				require.EqualError(t, err, "error loading the subscription")
			},
		},
		{
			name: "subscription loaded successfully",
			setupMock: func() {
				mscalendar.client = mockClient
				mscalendar.actingUser = GetMockUser(model.NewString(MockActingUserRemoteID), nil, MockActingUserID, nil)
				mscalendar.actingUser.Settings.EventSubscriptionID = MockEventSubscriptionID
				mockPluginAPI.EXPECT().GetMattermostUser(MockActingUserID).Return(&model.User{}, nil)
				mockStore.EXPECT().LoadSubscription(MockEventSubscriptionID).Return(expectedSubscription, nil).Times(1)
			},
			assertion: func(sub *store.Subscription, err error) {
				require.NoError(t, err)
				require.Equal(t, expectedSubscription, sub)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			sub, err := mscalendar.LoadMyEventSubscription()

			tt.assertion(sub, err)
		})
	}
}

func TestListRemoteSubscriptions(t *testing.T) {
	mscalendar, mockStore, _, _, _, mockClient, _ := GetMockSetup(t)

	tests := []struct {
		name      string
		setupMock func()
		assertion func(subs []*remote.Subscription, err error)
	}{
		{
			name: "error filtering with user",
			setupMock: func() {
				mscalendar.client = nil
				mscalendar.actingUser = GetMockUser(nil, nil, MockActingUserID, nil)
				mockStore.EXPECT().LoadUser(MockActingUserID).Return(nil, errors.New("error filtering the user")).Times(1)
			},
			assertion: func(subs []*remote.Subscription, err error) {
				require.ErrorContains(t, err, "error filtering the user")
			},
		},
		{
			name: "error listing the subscription",
			setupMock: func() {
				mscalendar.client = mockClient
				mscalendar.actingUser = GetMockUser(model.NewString(MockActingUserRemoteID), nil, MockActingUserID, nil)
				mscalendar.actingUser.Settings.EventSubscriptionID = MockEventSubscriptionID
				mockClient.EXPECT().ListSubscriptions().Return(nil, errors.New("error listing the subscriptions"))
			},
			assertion: func(subs []*remote.Subscription, err error) {
				require.EqualError(t, err, "error listing the subscriptions")
			},
		},
		{
			name: "subscriptions listed successfully",
			setupMock: func() {
				mscalendar.client = mockClient
				mscalendar.actingUser = GetMockUser(model.NewString(MockActingUserRemoteID), nil, MockActingUserID, nil)
				mscalendar.actingUser.Settings.EventSubscriptionID = MockEventSubscriptionID
				mockClient.EXPECT().ListSubscriptions().Return([]*remote.Subscription{{ID: "mockSubscription1"}, {ID: "mockSubscription2"}}, nil)
			},
			assertion: func(subs []*remote.Subscription, err error) {
				require.NoError(t, err)
				require.Equal(t, []*remote.Subscription{{ID: "mockSubscription1"}, {ID: "mockSubscription2"}}, subs)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			subs, err := mscalendar.ListRemoteSubscriptions()

			tt.assertion(subs, err)
		})
	}
}

func TestRenewMyEventSubscription(t *testing.T) {
	mscalendar, mockStore, _, mockRemote, mockPluginAPI, mockClient, _ := GetMockSetup(t)

	tests := []struct {
		name      string
		setupMock func()
		assertion func(subs *store.Subscription, err error)
	}{
		{
			name: "error filtering with the user",
			setupMock: func() {
				mscalendar.client = nil
				mscalendar.actingUser = GetMockUser(nil, nil, MockActingUserID, nil)
				mockStore.EXPECT().LoadUser(MockActingUserID).Return(nil, errors.New("error filtering the user")).Times(1)
			},
			assertion: func(subs *store.Subscription, err error) {
				require.ErrorContains(t, err, "error filtering the user")
			},
		},
		{
			name: "no subscriptions present",
			setupMock: func() {
				mockPluginAPI.EXPECT().GetMattermostUser(MockActingUserID).Return(&model.User{}, nil)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
				mscalendar.actingUser = GetMockUser(model.NewString(MockActingUserRemoteID), nil, MockActingUserID, nil)
				mscalendar.actingUser.Settings.EventSubscriptionID = ""
			},
			assertion: func(subs *store.Subscription, err error) {
				require.NoError(t, err)
				require.Nil(t, subs)
			},
		},
		{
			name: "error loading the subscription",
			setupMock: func() {
				mockPluginAPI.EXPECT().GetMattermostUser(MockActingUserID).Return(&model.User{}, nil)
				mockRemote.EXPECT().MakeUserClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
				mscalendar.actingUser = GetMockUser(model.NewString(MockActingUserRemoteID), nil, MockActingUserID, nil)
				mscalendar.actingUser.Settings.EventSubscriptionID = MockEventSubscriptionID
				mockStore.EXPECT().LoadSubscription(MockEventSubscriptionID).Return(nil, errors.New("some error occurred while loading the subscription"))
			},
			assertion: func(subs *store.Subscription, err error) {
				require.Error(t, err)
				require.EqualError(t, err, "error loading subscription: some error occurred while loading the subscription")
			},
		},
		{
			name: "error renewing the subscription",
			setupMock: func() {
				mscalendar.client = mockClient
				mscalendar.actingUser = GetMockUser(model.NewString(MockActingUserRemoteID), nil, MockActingUserID, nil)
				mscalendar.actingUser.Settings.EventSubscriptionID = MockEventSubscriptionID
				mockStore.EXPECT().LoadSubscription(MockEventSubscriptionID).Return(&store.Subscription{Remote: &remote.Subscription{}}, nil)
				mockClient.EXPECT().RenewSubscription(gomock.Any(), MockActingUserRemoteID, &remote.Subscription{}).Return(nil, errors.New("The object was not found")).Times(1)
				mockStore.EXPECT().DeleteUserSubscription(gomock.Any(), MockEventSubscriptionID).Return(errors.New("error deleting the subscription")).Times(1)
			},
			assertion: func(subs *store.Subscription, err error) {
				require.Error(t, err)
				require.EqualError(t, err, "error deleting the subscription")
			},
		},
		{
			name: "successfully renew the event subscription",
			setupMock: func() {
				mscalendar.client = mockClient
				mscalendar.actingUser = GetMockUser(model.NewString(MockActingUserRemoteID), nil, MockActingUserID, nil)
				mscalendar.actingUser.Settings.EventSubscriptionID = MockEventSubscriptionID
				mockStore.EXPECT().LoadSubscription(MockEventSubscriptionID).Return(&store.Subscription{Remote: &remote.Subscription{}}, nil).Times(2)
				mockClient.EXPECT().RenewSubscription(gomock.Any(), MockActingUserRemoteID, &remote.Subscription{}).Return(&remote.Subscription{}, nil).Times(1)
				mockStore.EXPECT().StoreUserSubscription(gomock.Any(), gomock.Any()).Return(nil)
			},
			assertion: func(subs *store.Subscription, err error) {
				require.NoError(t, err)
				require.Equal(t, &store.Subscription{Remote: &remote.Subscription{}}, subs)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			subs, err := mscalendar.RenewMyEventSubscription()

			tt.assertion(subs, err)
		})
	}
}

func TestDeleteMyEventSubscription(t *testing.T) {
	mscalendar, mockStore, _, _, mockPluginAPI, mockClient, _ := GetMockSetup(t)

	tests := []struct {
		name      string
		setupMock func()
		assertion func(err error)
	}{
		{
			name: "error filtering with the user",
			setupMock: func() {
				mscalendar.client = nil
				mscalendar.actingUser = GetMockUser(nil, nil, MockActingUserID, nil)
				mockStore.EXPECT().LoadUser(MockActingUserID).Return(nil, errors.New("error filtering the user")).Times(1)
			},
			assertion: func(err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "error filtering the user")
			},
		},
		{
			name: "error loading the subscription",
			setupMock: func() {
				mscalendar.actingUser = &User{User: &store.User{Remote: &remote.User{ID: MockActingUserRemoteID}}, MattermostUserID: MockActingUserID}
				mscalendar.actingUser.Settings.EventSubscriptionID = MockEventSubscriptionID
				mockPluginAPI.EXPECT().GetMattermostUser(MockActingUserID).Return(&model.User{}, nil)
				mockStore.EXPECT().LoadSubscription(MockEventSubscriptionID).Return(nil, errors.New("some error occurred while loading the subscription")).Times(1)
			},
			assertion: func(err error) {
				require.Error(t, err)
				require.EqualError(t, err, "error loading subscription: some error occurred while loading the subscription")
			},
		},
		{
			name: "error deleting the subscription in DeleteOrphanedSubscription",
			setupMock: func() {
				mscalendar.actingUser = &User{User: &store.User{Remote: &remote.User{ID: MockActingUserRemoteID}}, MattermostUserID: MockActingUserID}
				mscalendar.actingUser.Settings.EventSubscriptionID = MockEventSubscriptionID
				mockPluginAPI.EXPECT().GetMattermostUser(MockActingUserID).Return(&model.User{}, nil).Times(1)
				mockStore.EXPECT().LoadSubscription(MockEventSubscriptionID).Return(&store.Subscription{Remote: &remote.Subscription{}}, nil).Times(1)
				mscalendar.client = mockClient
				mockClient.EXPECT().DeleteSubscription(&remote.Subscription{}).Return(errors.New("some error occured")).Times(1)
			},
			assertion: func(err error) {
				require.Error(t, err)
				require.EqualError(t, err, "failed to delete subscription : some error occured")
			},
		},
		{
			name: "error deleting the user subscription",
			setupMock: func() {
				mscalendar.actingUser = &User{User: &store.User{Remote: &remote.User{ID: MockActingUserRemoteID}}, MattermostUserID: MockActingUserID}
				mscalendar.actingUser.Settings.EventSubscriptionID = MockEventSubscriptionID
				mockPluginAPI.EXPECT().GetMattermostUser(MockActingUserID).Return(&model.User{}, nil).Times(1)
				mockStore.EXPECT().LoadSubscription(MockEventSubscriptionID).Return(&store.Subscription{Remote: &remote.Subscription{}}, nil).Times(1)
				mscalendar.client = mockClient
				mockClient.EXPECT().DeleteSubscription(&remote.Subscription{}).Return(nil).Times(1)
				mockStore.EXPECT().DeleteUserSubscription(gomock.Any(), MockEventSubscriptionID).Return(errors.New("error deleting the user subscription"))
			},
			assertion: func(err error) {
				require.Error(t, err)
				require.EqualError(t, err, "failed to delete subscription testEventSubscriptionID: error deleting the user subscription")
			},
		},
		{
			name: "event subscription deleted successfully",
			setupMock: func() {
				mscalendar.actingUser = &User{User: &store.User{Remote: &remote.User{ID: MockActingUserRemoteID}}, MattermostUserID: MockActingUserID}
				mscalendar.actingUser.Settings.EventSubscriptionID = MockEventSubscriptionID
				mockPluginAPI.EXPECT().GetMattermostUser(MockActingUserID).Return(&model.User{}, nil).Times(1)
				mockStore.EXPECT().LoadSubscription(MockEventSubscriptionID).Return(&store.Subscription{Remote: &remote.Subscription{}}, nil).Times(1)
				mscalendar.client = mockClient
				mockClient.EXPECT().DeleteSubscription(&remote.Subscription{}).Return(nil).Times(1)
				mockStore.EXPECT().DeleteUserSubscription(gomock.Any(), MockEventSubscriptionID).Return(nil)
			},
			assertion: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := mscalendar.DeleteMyEventSubscription()

			tt.assertion(err)
		})
	}
}

func TestDeleteOrphanedSubscription(t *testing.T) {
	mscalendar, mockStore, _, _, _, mockClient, _ := GetMockSetup(t)

	tests := []struct {
		name      string
		setupMock func()
		assertion func(err error)
	}{
		{
			name: "error filtering with the user",
			setupMock: func() {
				mscalendar.client = nil
				mscalendar.actingUser = GetMockUser(nil, nil, MockActingUserID, nil)
				mockStore.EXPECT().LoadUser(MockActingUserID).Return(nil, errors.New("error filtering the user")).Times(1)
			},
			assertion: func(err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "error filtering the user")
			},
		},
		{
			name: "error deleting the subscription",
			setupMock: func() {
				mscalendar.client = mockClient
				mscalendar.actingUser = &User{User: &store.User{Remote: &remote.User{ID: MockActingUserRemoteID}}, MattermostUserID: MockActingUserID}
				mockClient.EXPECT().DeleteSubscription(gomock.Any()).Return(errors.New("error deleting the subscription"))
			},
			assertion: func(err error) {
				require.Error(t, err)
				require.EqualError(t, err, "failed to delete subscription : error deleting the subscription")
			},
		},
		{
			name: "subscription deleted sucessfully",
			setupMock: func() {
				mscalendar.client = mockClient
				mscalendar.actingUser = &User{User: &store.User{Remote: &remote.User{ID: MockActingUserRemoteID}}, MattermostUserID: MockActingUserID}
				mockClient.EXPECT().DeleteSubscription(gomock.Any()).Return(nil)
			},
			assertion: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			subscription := GetMockSubscription()

			err := mscalendar.DeleteOrphanedSubscription(subscription)

			tt.assertion(err)
		})
	}
}
