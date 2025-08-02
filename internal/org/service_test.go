package org

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Gkemhcs/kavach-backend/internal/iam"
	iam_db "github.com/Gkemhcs/kavach-backend/internal/iam/gen"
	orgdb "github.com/Gkemhcs/kavach-backend/internal/org/gen"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockQuerier is a mock implementation of the database interface
type MockQuerier struct {
	mock.Mock
}

func (m *MockQuerier) CreateOrganization(ctx context.Context, arg orgdb.CreateOrganizationParams) (orgdb.Organization, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(orgdb.Organization), args.Error(1)
}

func (m *MockQuerier) ListOrganizationsByOwner(ctx context.Context, ownerID uuid.UUID) ([]orgdb.Organization, error) {
	args := m.Called(ctx, ownerID)
	return args.Get(0).([]orgdb.Organization), args.Error(1)
}

func (m *MockQuerier) GetOrganizationByID(ctx context.Context, id uuid.UUID) (orgdb.Organization, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(orgdb.Organization), args.Error(1)
}

func (m *MockQuerier) GetOrganizationByName(ctx context.Context, name string) (orgdb.Organization, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(orgdb.Organization), args.Error(1)
}

func (m *MockQuerier) UpdateOrganization(ctx context.Context, arg orgdb.UpdateOrganizationParams) (orgdb.Organization, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(orgdb.Organization), args.Error(1)
}

func (m *MockQuerier) DeleteOrganization(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockIamService is a mock implementation of the IAM service
type MockIamService struct {
	mock.Mock
}

func (m *MockIamService) CreateRoleBinding(ctx context.Context, req iam.CreateRoleBindingRequest) (*iam_db.RoleBinding, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*iam_db.RoleBinding), args.Error(1)
}

func (m *MockIamService) ListAccessibleOrganizations(ctx context.Context, userID string) ([]iam_db.ListAccessibleOrganizationsRow, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]iam_db.ListAccessibleOrganizationsRow), args.Error(1)
}

// MockEnforcer is a mock implementation of the policy enforcer
type MockEnforcer struct {
	mock.Mock
}

func (m *MockEnforcer) GrantRoleWithPermissions(userID, role, resource string) error {
	args := m.Called(userID, role, resource)
	return args.Error(0)
}

func (m *MockEnforcer) CheckPermission(userID, action, resource string) (bool, error) {
	args := m.Called(userID, action, resource)
	return args.Bool(0), args.Error(1)
}

// Test helper functions
func createTestOrganization() orgdb.Organization {
	now := time.Now()
	return orgdb.Organization{
		ID:          uuid.New(),
		Name:        "Test Organization",
		Description: "Test organization description",
		OwnerID:     uuid.New(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func createTestUserID() uuid.UUID {
	return uuid.New()
}

// TestCreateOrganization tests the CreateOrganization method
func TestCreateOrganization(t *testing.T) {
	tests := []struct {
		name          string
		request       CreateOrganizationRequest
		setupMocks    func(*MockQuerier, *MockIamService, *MockEnforcer)
		expectedError bool
		errorMessage  string
	}{
		{
			name: "Success - Valid organization creation",
			request: CreateOrganizationRequest{
				Name:        "Test Org",
				Description: "Test description",
				UserID:      createTestUserID().String(),
			},
			setupMocks: func(mockRepo *MockQuerier, mockIam *MockIamService, mockEnforcer *MockEnforcer) {
				expectedOrg := createTestOrganization()
				expectedOrg.Name = "Test Org"
				expectedOrg.Description = "Test description"

				mockRepo.On("CreateOrganization", mock.Anything, mock.AnythingOfType("orgdb.CreateOrganizationParams")).
					Return(expectedOrg, nil)

				mockIam.On("CreateRoleBinding", mock.Anything, mock.AnythingOfType("iam.CreateRoleBindingRequest")).
					Return(&iam_db.RoleBinding{}, nil)

				mockEnforcer.On("GrantRoleWithPermissions", mock.AnythingOfType("string"), "owner", mock.AnythingOfType("string")).
					Return(nil)
			},
			expectedError: false,
		},
		{
			name: "Error - Invalid UUID",
			request: CreateOrganizationRequest{
				Name:        "Test Org",
				Description: "Test description",
				UserID:      "invalid-uuid",
			},
			setupMocks: func(mockRepo *MockQuerier, mockIam *MockIamService, mockEnforcer *MockEnforcer) {
				// No mocks needed as it should fail before reaching the database
			},
			expectedError: true,
			errorMessage:  "internal server error",
		},
		{
			name: "Error - Database error",
			request: CreateOrganizationRequest{
				Name:        "Test Org",
				Description: "Test description",
				UserID:      createTestUserID().String(),
			},
			setupMocks: func(mockRepo *MockQuerier, mockIam *MockIamService, mockEnforcer *MockEnforcer) {
				mockRepo.On("CreateOrganization", mock.Anything, mock.AnythingOfType("orgdb.CreateOrganizationParams")).
					Return(orgdb.Organization{}, errors.New("database error"))
			},
			expectedError: true,
			errorMessage:  "internal server error",
		},
		{
			name: "Error - IAM service error",
			request: CreateOrganizationRequest{
				Name:        "Test Org",
				Description: "Test description",
				UserID:      createTestUserID().String(),
			},
			setupMocks: func(mockRepo *MockQuerier, mockIam *MockIamService, mockEnforcer *MockEnforcer) {
				expectedOrg := createTestOrganization()
				mockRepo.On("CreateOrganization", mock.Anything, mock.AnythingOfType("orgdb.CreateOrganizationParams")).
					Return(expectedOrg, nil)

				mockIam.On("CreateRoleBinding", mock.Anything, mock.AnythingOfType("iam.CreateRoleBindingRequest")).
					Return(nil, errors.New("iam error"))
			},
			expectedError: true,
			errorMessage:  "iam error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockRepo := new(MockQuerier)
			mockIam := new(MockIamService)
			mockEnforcer := new(MockEnforcer)
			logger := logrus.New()

			// Setup service
			service := NewOrganizationService(mockRepo, logger, mockIam, mockEnforcer)

			// Setup mocks
			tt.setupMocks(mockRepo, mockIam, mockEnforcer)

			// Execute test
			result, err := service.CreateOrganization(context.Background(), tt.request)

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.request.Name, result.Name)
				assert.Equal(t, tt.request.Description, result.Description)
			}

			// Verify mocks
			mockRepo.AssertExpectations(t)
			mockIam.AssertExpectations(t)
			mockEnforcer.AssertExpectations(t)
		})
	}
}

// TestListOrganizations tests the ListOrganizations method
func TestListOrganizations(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		setupMocks    func(*MockQuerier)
		expectedCount int
		expectedError bool
	}{
		{
			name:   "Success - List organizations",
			userID: createTestUserID().String(),
			setupMocks: func(mockRepo *MockQuerier) {
				orgs := []orgdb.Organization{
					createTestOrganization(),
					createTestOrganization(),
				}
				mockRepo.On("ListOrganizationsByOwner", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(orgs, nil)
			},
			expectedCount: 2,
			expectedError: false,
		},
		{
			name:   "Error - Invalid UUID",
			userID: "invalid-uuid",
			setupMocks: func(mockRepo *MockQuerier) {
				// No mocks needed as it should fail before reaching the database
			},
			expectedCount: 0,
			expectedError: true,
		},
		{
			name:   "Error - Database error",
			userID: createTestUserID().String(),
			setupMocks: func(mockRepo *MockQuerier) {
				mockRepo.On("ListOrganizationsByOwner", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return([]orgdb.Organization{}, errors.New("database error"))
			},
			expectedCount: 0,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockRepo := new(MockQuerier)
			mockIam := new(MockIamService)
			mockEnforcer := new(MockEnforcer)
			logger := logrus.New()

			// Setup service
			service := NewOrganizationService(mockRepo, logger, mockIam, mockEnforcer)

			// Setup mocks
			tt.setupMocks(mockRepo)

			// Execute test
			result, err := service.ListOrganizations(context.Background(), tt.userID)

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.expectedCount)
			}

			// Verify mocks
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestGetOrganization tests the GetOrganization method
func TestGetOrganization(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		orgID         string
		setupMocks    func(*MockQuerier, *MockEnforcer)
		expectedError bool
	}{
		{
			name:   "Success - Get organization",
			userID: createTestUserID().String(),
			orgID:  createTestUserID().String(),
			setupMocks: func(mockRepo *MockQuerier, mockEnforcer *MockEnforcer) {
				expectedOrg := createTestOrganization()
				mockRepo.On("GetOrganizationByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(expectedOrg, nil)

				mockEnforcer.On("CheckPermission", mock.AnythingOfType("string"), "read", mock.AnythingOfType("string")).
					Return(true, nil)
			},
			expectedError: false,
		},
		{
			name:   "Error - Invalid org UUID",
			userID: createTestUserID().String(),
			orgID:  "invalid-uuid",
			setupMocks: func(mockRepo *MockQuerier, mockEnforcer *MockEnforcer) {
				// No mocks needed as it should fail before reaching the database
			},
			expectedError: true,
		},
		{
			name:   "Error - Permission denied",
			userID: createTestUserID().String(),
			orgID:  createTestUserID().String(),
			setupMocks: func(mockRepo *MockQuerier, mockEnforcer *MockEnforcer) {
				mockEnforcer.On("CheckPermission", mock.AnythingOfType("string"), "read", mock.AnythingOfType("string")).
					Return(false, nil)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockRepo := new(MockQuerier)
			mockIam := new(MockIamService)
			mockEnforcer := new(MockEnforcer)
			logger := logrus.New()

			// Setup service
			service := NewOrganizationService(mockRepo, logger, mockIam, mockEnforcer)

			// Setup mocks
			tt.setupMocks(mockRepo, mockEnforcer)

			// Execute test
			result, err := service.GetOrganization(context.Background(), tt.userID, tt.orgID)

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			// Verify mocks
			mockRepo.AssertExpectations(t)
			mockEnforcer.AssertExpectations(t)
		})
	}
}

// TestUpdateOrganization tests the UpdateOrganization method
func TestUpdateOrganization(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		orgID         string
		request       UpdateOrganizationRequest
		setupMocks    func(*MockQuerier, *MockEnforcer)
		expectedError bool
	}{
		{
			name:   "Success - Update organization",
			userID: createTestUserID().String(),
			orgID:  createTestUserID().String(),
			request: UpdateOrganizationRequest{
				Name:        "Updated Org",
				Description: "Updated description",
				UserID:      createTestUserID().String(),
			},
			setupMocks: func(mockRepo *MockQuerier, mockEnforcer *MockEnforcer) {
				expectedOrg := createTestOrganization()
				expectedOrg.Name = "Updated Org"
				expectedOrg.Description = "Updated description"

				mockEnforcer.On("CheckPermission", mock.AnythingOfType("string"), "update", mock.AnythingOfType("string")).
					Return(true, nil)

				mockRepo.On("UpdateOrganization", mock.Anything, mock.AnythingOfType("orgdb.UpdateOrganizationParams")).
					Return(expectedOrg, nil)
			},
			expectedError: false,
		},
		{
			name:   "Error - Permission denied",
			userID: createTestUserID().String(),
			orgID:  createTestUserID().String(),
			request: UpdateOrganizationRequest{
				Name:        "Updated Org",
				Description: "Updated description",
				UserID:      createTestUserID().String(),
			},
			setupMocks: func(mockRepo *MockQuerier, mockEnforcer *MockEnforcer) {
				mockEnforcer.On("CheckPermission", mock.AnythingOfType("string"), "update", mock.AnythingOfType("string")).
					Return(false, nil)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockRepo := new(MockQuerier)
			mockIam := new(MockIamService)
			mockEnforcer := new(MockEnforcer)
			logger := logrus.New()

			// Setup service
			service := NewOrganizationService(mockRepo, logger, mockIam, mockEnforcer)

			// Setup mocks
			tt.setupMocks(mockRepo, mockEnforcer)

			// Execute test
			result, err := service.UpdateOrganization(context.Background(), tt.userID, tt.orgID, tt.request)

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.request.Name, result.Name)
				assert.Equal(t, tt.request.Description, result.Description)
			}

			// Verify mocks
			mockRepo.AssertExpectations(t)
			mockEnforcer.AssertExpectations(t)
		})
	}
}

// TestDeleteOrganization tests the DeleteOrganization method
func TestDeleteOrganization(t *testing.T) {
	tests := []struct {
		name          string
		orgID         string
		setupMocks    func(*MockQuerier)
		expectedError bool
	}{
		{
			name:  "Success - Delete organization",
			orgID: createTestUserID().String(),
			setupMocks: func(mockRepo *MockQuerier) {
				mockRepo.On("DeleteOrganization", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(nil)
			},
			expectedError: false,
		},
		{
			name:  "Error - Invalid UUID",
			orgID: "invalid-uuid",
			setupMocks: func(mockRepo *MockQuerier) {
				// No mocks needed as it should fail before reaching the database
			},
			expectedError: true,
		},
		{
			name:  "Error - Database error",
			orgID: createTestUserID().String(),
			setupMocks: func(mockRepo *MockQuerier) {
				mockRepo.On("DeleteOrganization", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockRepo := new(MockQuerier)
			mockIam := new(MockIamService)
			mockEnforcer := new(MockEnforcer)
			logger := logrus.New()

			// Setup service
			service := NewOrganizationService(mockRepo, logger, mockIam, mockEnforcer)

			// Setup mocks
			tt.setupMocks(mockRepo)

			// Execute test
			err := service.DeleteOrganization(context.Background(), tt.orgID)

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify mocks
			mockRepo.AssertExpectations(t)
		})
	}
}
