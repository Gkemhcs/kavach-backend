package secretgroup

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	userdb "github.com/Gkemhcs/kavach-backend/internal/auth/gen"
	"github.com/Gkemhcs/kavach-backend/internal/authz"
	groupsdb "github.com/Gkemhcs/kavach-backend/internal/groups/gen"
	"github.com/Gkemhcs/kavach-backend/internal/iam"
	iam_db "github.com/Gkemhcs/kavach-backend/internal/iam/gen"
	secretgroupdb "github.com/Gkemhcs/kavach-backend/internal/secretgroup/gen"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//go:embed test_data/*.json
var testDataFS embed.FS

// TestData represents the structure of test data files
type TestData struct {
	TestCases []TestCase `json:"test_cases"`
}

// TestCase represents a single test case
type TestCase struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
	Expected    ExpectedResult         `json:"expected"`
	MockSetup   MockSetup              `json:"mock_setup"`
}

// ExpectedResult represents the expected outcome of a test
type ExpectedResult struct {
	Success bool        `json:"success"`
	Error   interface{} `json:"error"`
	// Additional fields for specific test types
	SecretGroup  interface{}              `json:"secret_group,omitempty"`
	SecretGroups []map[string]interface{} `json:"secret_groups,omitempty"`
}

// MockSetup represents the mock configuration for a test case
type MockSetup struct {
	SecretGroupRepo MockConfig   `json:"secret_group_repo,omitempty"`
	IamService      MockConfig   `json:"iam_service,omitempty"`
	PolicyEnforcer  []MockConfig `json:"policy_enforcer,omitempty"`
}

// MockConfig represents configuration for a specific mock
type MockConfig struct {
	Method string                 `json:"method"`
	Return map[string]interface{} `json:"return"`
}

// SecretGroupServiceTestSuite is the test suite for SecretGroupService
type SecretGroupServiceTestSuite struct {
	suite.Suite
	service            *SecretGroupService
	mockRepo           *MockSecretGroupRepository
	mockIamRepo        *iam.MockIamRepository
	mockUserResolver   *MockUserResolver
	mockGroupResolver  *MockUserGroupResolver
	mockPolicyEnforcer *MockPolicyEnforcer
	iamService         *iam.IamService
	logger             *logrus.Logger
	ctx                context.Context
}

// MockUserResolver mocks the user resolver interface
type MockUserResolver struct {
	mock.Mock
}

// GetUserInfoByGithubUserName mocks the user resolution method
func (m *MockUserResolver) GetUserInfoByGithubUserName(ctx context.Context, username string) (*userdb.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userdb.User), args.Error(1)
}

// MockUserGroupResolver mocks the user group resolver interface
type MockUserGroupResolver struct {
	mock.Mock
}

// GetUserGroupByName mocks the user group resolution method
func (m *MockUserGroupResolver) GetUserGroupByName(ctx context.Context, groupName, orgID string) (*groupsdb.UserGroup, error) {
	args := m.Called(ctx, groupName, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*groupsdb.UserGroup), args.Error(1)
}

// MockPolicyEnforcer wraps the authz mock enforcer
type MockPolicyEnforcer struct {
	*authz.MockEnforcer
}

// SetupSuite initializes the test suite
func (suite *SecretGroupServiceTestSuite) SetupSuite() {
	suite.logger = logrus.New()
	suite.logger.SetOutput(io.Discard) // Disable logging to stdout
	suite.ctx = context.Background()
}

// SetupTest initializes each test
func (suite *SecretGroupServiceTestSuite) SetupTest() {
	suite.mockRepo = &MockSecretGroupRepository{}
	suite.mockIamRepo = &iam.MockIamRepository{}
	suite.mockUserResolver = &MockUserResolver{}
	suite.mockGroupResolver = &MockUserGroupResolver{}
	suite.mockPolicyEnforcer = &MockPolicyEnforcer{
		MockEnforcer: &authz.MockEnforcer{},
	}
	suite.iamService = iam.NewIamService(
		suite.mockIamRepo,
		suite.mockUserResolver,
		suite.mockGroupResolver,
		suite.logger,
		suite.mockPolicyEnforcer,
	)
	suite.service = NewSecretGroupService(
		suite.mockRepo,
		suite.logger,
		*suite.iamService,
		suite.mockPolicyEnforcer,
	)
}

// TearDownTest cleans up after each test
func (suite *SecretGroupServiceTestSuite) TearDownTest() {
	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockIamRepo.AssertExpectations(suite.T())
	suite.mockUserResolver.AssertExpectations(suite.T())
	suite.mockGroupResolver.AssertExpectations(suite.T())
	suite.mockPolicyEnforcer.MockEnforcer.AssertExpectations(suite.T())
	suite.mockRepo.ExpectedCalls = nil
	suite.mockIamRepo.ExpectedCalls = nil
	suite.mockUserResolver.ExpectedCalls = nil
	suite.mockGroupResolver.ExpectedCalls = nil
	suite.mockPolicyEnforcer.MockEnforcer.ExpectedCalls = nil
}

// loadTestData loads test data from embedded JSON files
func (suite *SecretGroupServiceTestSuite) loadTestData(filename string) *TestData {
	data, err := testDataFS.ReadFile("test_data/" + filename)
	require.NoError(suite.T(), err, "Failed to read test data file: %s", filename)

	var testData TestData
	err = json.Unmarshal(data, &testData)
	require.NoError(suite.T(), err, "Failed to unmarshal test data from: %s", filename)

	return &testData
}

// TestCreateSecretGroupWithData tests CreateSecretGroup with data-driven test cases
func (suite *SecretGroupServiceTestSuite) TestCreateSecretGroupWithData() {
	testData := suite.loadTestData("create_secret_group_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupCreateSecretGroupMocks(tc.MockSetup)

			// Build request from test data
			req := suite.buildCreateSecretGroupRequest(tc.Input)

			// Call the service method
			result, err := suite.service.CreateSecretGroup(suite.ctx, req)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedGroup := tc.Expected.SecretGroup.(map[string]interface{})
				assert.Equal(suite.T(), expectedGroup["name"].(string), result.Name, "Secret group name mismatch")
				assert.Equal(suite.T(), expectedGroup["organization_id"].(string), result.OrganizationID.String(), "Organization ID mismatch")
				if expectedGroup["description"] != nil && result.Description.Valid {
					assert.Equal(suite.T(), expectedGroup["description"].(string), result.Description.String, "Secret group description mismatch")
				}
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				if tc.Expected.Error != nil {
					expectedError := strings.ToLower(fmt.Sprintf("%v", tc.Expected.Error))
					actualError := strings.ToLower(err.Error())
					require.Contains(suite.T(), actualError, expectedError, "Error message mismatch")
				}
				// Validate that downstream methods were NOT called in error scenarios
				suite.validateNegativeMocks(tc.Name)
			}
		})
	}
}

// TestListSecretGroupsWithData tests ListSecretGroups with data-driven test cases
func (suite *SecretGroupServiceTestSuite) TestListSecretGroupsWithData() {
	testData := suite.loadTestData("list_secret_groups_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupListSecretGroupsMocks(tc.MockSetup)

			// Get input parameters
			userID := tc.Input["user_id"].(string)
			orgID := tc.Input["org_id"].(string)

			// Call the service method
			result, err := suite.service.ListSecretGroups(suite.ctx, userID, orgID)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedGroups := tc.Expected.SecretGroups
				assert.Equal(suite.T(), len(expectedGroups), len(result), "Number of secret groups mismatch")
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				if tc.Expected.Error != nil {
					expectedError := strings.ToLower(fmt.Sprintf("%v", tc.Expected.Error))
					actualError := strings.ToLower(err.Error())
					require.Contains(suite.T(), actualError, expectedError, "Error message mismatch")
				}
			}
		})
	}
}

// TestListMySecretGroupsWithData tests ListMySecretGroups with data-driven test cases
func (suite *SecretGroupServiceTestSuite) TestListMySecretGroupsWithData() {
	testData := suite.loadTestData("list_my_secret_groups_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupListMySecretGroupsMocks(tc.MockSetup)

			// Get input parameters
			orgID := tc.Input["org_id"].(string)
			userID := tc.Input["user_id"].(string)

			// Call the service method
			result, err := suite.service.ListMySecretGroups(suite.ctx, orgID, userID)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedGroups := tc.Expected.SecretGroups
				assert.Equal(suite.T(), len(expectedGroups), len(result), "Number of secret groups mismatch")

				for i, expectedGroup := range expectedGroups {
					if i < len(result) {
						assert.Equal(suite.T(), expectedGroup["name"].(string), result[i].Name, "Secret group name mismatch")
						assert.Equal(suite.T(), expectedGroup["organization_name"].(string), result[i].OrganizationName, "Organization name mismatch")
						assert.Equal(suite.T(), expectedGroup["role"].(string), string(result[i].Role), "Role mismatch")
					}
				}
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				if tc.Expected.Error != nil {
					expectedError := strings.ToLower(fmt.Sprintf("%v", tc.Expected.Error))
					actualError := strings.ToLower(err.Error())
					require.Contains(suite.T(), actualError, expectedError, "Error message mismatch")
				}
			}
		})
	}
}

// TestGetSecretGroupWithData tests GetSecretGroup with data-driven test cases
func (suite *SecretGroupServiceTestSuite) TestGetSecretGroupWithData() {
	testData := suite.loadTestData("get_secret_group_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupGetSecretGroupMocks(tc.MockSetup)

			// Get input parameters
			userID := tc.Input["user_id"].(string)
			orgID := tc.Input["org_id"].(string)
			groupID := tc.Input["group_id"].(string)

			// Call the service method
			result, err := suite.service.GetSecretGroup(suite.ctx, userID, orgID, groupID)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedGroup := tc.Expected.SecretGroup.(map[string]interface{})
				assert.Equal(suite.T(), expectedGroup["name"].(string), result.Name, "Secret group name mismatch")
				assert.Equal(suite.T(), expectedGroup["organization_id"].(string), result.OrganizationID.String(), "Organization ID mismatch")
				if expectedGroup["description"] != nil && result.Description.Valid {
					assert.Equal(suite.T(), expectedGroup["description"].(string), result.Description.String, "Secret group description mismatch")
				}
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				if tc.Expected.Error != nil {
					expectedError := strings.ToLower(fmt.Sprintf("%v", tc.Expected.Error))
					actualError := strings.ToLower(err.Error())
					require.Contains(suite.T(), actualError, expectedError, "Error message mismatch")
				}
			}
		})
	}
}

// TestGetSecretGroupByNameWithData tests GetSecretGroupByName with data-driven test cases
func (suite *SecretGroupServiceTestSuite) TestGetSecretGroupByNameWithData() {
	testData := suite.loadTestData("get_secret_group_by_name_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupGetSecretGroupByNameMocks(tc.MockSetup)

			// Get input parameters
			orgID := tc.Input["org_id"].(string)
			groupName := tc.Input["group_name"].(string)

			// Call the service method
			result, err := suite.service.GetSecretGroupByName(suite.ctx, orgID, groupName)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedGroup := tc.Expected.SecretGroup.(map[string]interface{})
				assert.Equal(suite.T(), expectedGroup["name"].(string), result.Name, "Secret group name mismatch")
				assert.Equal(suite.T(), expectedGroup["organization_id"].(string), result.OrganizationID.String(), "Organization ID mismatch")
				if expectedGroup["description"] != nil && result.Description.Valid {
					assert.Equal(suite.T(), expectedGroup["description"].(string), result.Description.String, "Secret group description mismatch")
				}
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				if tc.Expected.Error != nil {
					expectedError := strings.ToLower(fmt.Sprintf("%v", tc.Expected.Error))
					actualError := strings.ToLower(err.Error())
					require.Contains(suite.T(), actualError, expectedError, "Error message mismatch")
				}
			}
		})
	}
}

// TestUpdateSecretGroupWithData tests UpdateSecretGroup with data-driven test cases
func (suite *SecretGroupServiceTestSuite) TestUpdateSecretGroupWithData() {
	testData := suite.loadTestData("update_secret_group_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupUpdateSecretGroupMocks(tc.MockSetup)

			// Get input parameters
			userID := tc.Input["user_id"].(string)
			orgID := tc.Input["org_id"].(string)
			groupID := tc.Input["group_id"].(string)

			// Build request from test data
			req := suite.buildUpdateSecretGroupRequest(tc.Input)

			// Call the service method
			result, err := suite.service.UpdateSecretGroup(suite.ctx, userID, orgID, groupID, req)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedGroup := tc.Expected.SecretGroup.(map[string]interface{})
				assert.Equal(suite.T(), expectedGroup["name"].(string), result.Name, "Secret group name mismatch")
				if expectedGroup["description"] != nil && result.Description.Valid {
					assert.Equal(suite.T(), expectedGroup["description"].(string), result.Description.String, "Secret group description mismatch")
				}
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				if tc.Expected.Error != nil {
					expectedError := strings.ToLower(fmt.Sprintf("%v", tc.Expected.Error))
					actualError := strings.ToLower(err.Error())
					require.Contains(suite.T(), actualError, expectedError, "Error message mismatch")
				}
			}
		})
	}
}

// TestDeleteSecretGroupWithData tests DeleteSecretGroup with data-driven test cases
func (suite *SecretGroupServiceTestSuite) TestDeleteSecretGroupWithData() {
	testData := suite.loadTestData("delete_secret_group_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupDeleteSecretGroupMocks(tc.MockSetup)

			// Get input parameters
			userID := tc.Input["user_id"].(string)
			orgID := tc.Input["org_id"].(string)
			groupID := tc.Input["group_id"].(string)

			// Call the service method
			err := suite.service.DeleteSecretGroup(suite.ctx, userID, orgID, groupID)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				if tc.Expected.Error != nil {
					expectedError := strings.ToLower(fmt.Sprintf("%v", tc.Expected.Error))
					actualError := strings.ToLower(err.Error())
					require.Contains(suite.T(), actualError, expectedError, "Error message mismatch")
				}
				// Validate that downstream methods were NOT called in error scenarios
				suite.validateNegativeMocks(tc.Name)
			}
		})
	}
}

// setupCreateSecretGroupMocks sets up mocks for CreateSecretGroup tests
func (suite *SecretGroupServiceTestSuite) setupCreateSecretGroupMocks(mockSetup MockSetup) {
	if mockSetup.SecretGroupRepo.Method != "" {
		suite.setupSecretGroupRepoMock(mockSetup.SecretGroupRepo)
	}

	if mockSetup.IamService.Method != "" {
		suite.setupIamServiceMock(mockSetup.IamService)
	}

	for _, policyConfig := range mockSetup.PolicyEnforcer {
		if policyConfig.Method != "" {
			suite.setupPolicyEnforcerMock(policyConfig)
		}
	}
}

// setupListSecretGroupsMocks sets up mocks for ListSecretGroups tests
func (suite *SecretGroupServiceTestSuite) setupListSecretGroupsMocks(mockSetup MockSetup) {
	if mockSetup.SecretGroupRepo.Method != "" {
		suite.setupSecretGroupRepoMock(mockSetup.SecretGroupRepo)
	}
}

// setupListMySecretGroupsMocks sets up mocks for ListMySecretGroups tests
func (suite *SecretGroupServiceTestSuite) setupListMySecretGroupsMocks(mockSetup MockSetup) {
	if mockSetup.IamService.Method != "" {
		suite.setupIamServiceMock(mockSetup.IamService)
	}
}

// setupGetSecretGroupMocks sets up mocks for GetSecretGroup tests
func (suite *SecretGroupServiceTestSuite) setupGetSecretGroupMocks(mockSetup MockSetup) {
	if mockSetup.SecretGroupRepo.Method != "" {
		suite.setupSecretGroupRepoMock(mockSetup.SecretGroupRepo)
	}
}

// setupGetSecretGroupByNameMocks sets up mocks for GetSecretGroupByName tests
func (suite *SecretGroupServiceTestSuite) setupGetSecretGroupByNameMocks(mockSetup MockSetup) {
	if mockSetup.SecretGroupRepo.Method != "" {
		suite.setupSecretGroupRepoMock(mockSetup.SecretGroupRepo)
	}
}

// setupUpdateSecretGroupMocks sets up mocks for UpdateSecretGroup tests
func (suite *SecretGroupServiceTestSuite) setupUpdateSecretGroupMocks(mockSetup MockSetup) {
	if mockSetup.SecretGroupRepo.Method != "" {
		suite.setupSecretGroupRepoMock(mockSetup.SecretGroupRepo)
	}
}

// setupDeleteSecretGroupMocks sets up mocks for DeleteSecretGroup tests
func (suite *SecretGroupServiceTestSuite) setupDeleteSecretGroupMocks(mockSetup MockSetup) {
	if mockSetup.SecretGroupRepo.Method != "" {
		suite.setupSecretGroupRepoMock(mockSetup.SecretGroupRepo)
	}

	if mockSetup.IamService.Method != "" {
		suite.setupIamServiceMock(mockSetup.IamService)
	}

	for _, policyConfig := range mockSetup.PolicyEnforcer {
		if policyConfig.Method != "" {
			suite.setupPolicyEnforcerMock(policyConfig)
		}
	}
}

// validateNegativeMocks validates that certain methods were NOT called when they shouldn't be
func (suite *SecretGroupServiceTestSuite) validateNegativeMocks(testCase string) {
	// CreateSecretGroup error cases - these should NOT call downstream methods
	if strings.Contains(testCase, "invalid_user_id") {
		// When user ID is invalid, IAM service and policy enforcer should NOT be called
		suite.mockRepo.AssertNotCalled(suite.T(), "CreateSecretGroup")
		suite.mockIamRepo.AssertNotCalled(suite.T(), "CreateRoleBinding")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "AddResourceOwner")
	}

	// CreateSecretGroup database error - should NOT call IAM service or policy enforcer
	if strings.Contains(testCase, "database_error") {
		suite.mockIamRepo.AssertNotCalled(suite.T(), "CreateRoleBinding")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "AddResourceOwner")
	}

	// CreateSecretGroup IAM error - should NOT call policy enforcer
	if strings.Contains(testCase, "iam_service_error") {
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "AddResourceOwner")
	}

	// DeleteSecretGroup error cases
	if strings.Contains(testCase, "invalid_group_id") {
		suite.mockRepo.AssertNotCalled(suite.T(), "DeleteSecretGroup")
		suite.mockIamRepo.AssertNotCalled(suite.T(), "DeleteRoleBinding")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "RemoveResource")
	}

	if strings.Contains(testCase, "group_not_found") {
		suite.mockIamRepo.AssertNotCalled(suite.T(), "DeleteRoleBinding")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "RemoveResource")
	}
}

// setupSecretGroupRepoMock sets up the secret group repository mock
func (suite *SecretGroupServiceTestSuite) setupSecretGroupRepoMock(config MockConfig) {
	switch config.Method {
	case "CreateSecretGroup":
		if config.Return["error"] != nil {
			suite.mockRepo.On("CreateSecretGroup", suite.ctx, mock.AnythingOfType("secretgroupdb.CreateSecretGroupParams")).
				Return(secretgroupdb.SecretGroup{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock secret group from test data
			groupData := config.Return["secret_group"].(map[string]interface{})
			group := secretgroupdb.SecretGroup{
				ID:             uuid.MustParse(groupData["id"].(string)),
				Name:           groupData["name"].(string),
				OrganizationID: uuid.MustParse(groupData["organization_id"].(string)),
			}
			if groupData["description"] != nil {
				desc := groupData["description"].(string)
				group.Description = sql.NullString{String: desc, Valid: true}
			}
			group.CreatedAt = time.Now()
			group.UpdatedAt = time.Now()

			suite.mockRepo.On("CreateSecretGroup", suite.ctx, mock.AnythingOfType("secretgroupdb.CreateSecretGroupParams")).
				Return(group, nil).Once()
		}
	case "ListSecretGroupsByOrg":
		if config.Return["error"] != nil {
			suite.mockRepo.On("ListSecretGroupsByOrg", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return([]secretgroupdb.SecretGroup{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock secret groups from test data
			groups := []secretgroupdb.SecretGroup{}
			if config.Return["secret_groups"] != nil {
				for _, group := range config.Return["secret_groups"].([]interface{}) {
					groupMap := group.(map[string]interface{})
					groupItem := secretgroupdb.SecretGroup{
						ID:             uuid.MustParse(groupMap["id"].(string)),
						Name:           groupMap["name"].(string),
						OrganizationID: uuid.MustParse(groupMap["organization_id"].(string)),
					}
					if groupMap["description"] != nil {
						desc := groupMap["description"].(string)
						groupItem.Description = sql.NullString{String: desc, Valid: true}
					}
					groupItem.CreatedAt = time.Now()
					groupItem.UpdatedAt = time.Now()
					groups = append(groups, groupItem)
				}
			}
			suite.mockRepo.On("ListSecretGroupsByOrg", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return(groups, nil).Once()
		}
	case "GetSecretGroupByID":
		if config.Return["error"] != nil {
			errorMsg := config.Return["error"].(string)
			if errorMsg == "sql: no rows in result set" {
				suite.mockRepo.On("GetSecretGroupByID", suite.ctx, mock.AnythingOfType("uuid.UUID")).
					Return(secretgroupdb.SecretGroup{}, sql.ErrNoRows).Once()
			} else {
				suite.mockRepo.On("GetSecretGroupByID", suite.ctx, mock.AnythingOfType("uuid.UUID")).
					Return(secretgroupdb.SecretGroup{}, errors.New(errorMsg)).Once()
			}
		} else {
			// Build mock secret group from test data
			groupData := config.Return["secret_group"].(map[string]interface{})
			group := secretgroupdb.SecretGroup{
				ID:             uuid.MustParse(groupData["id"].(string)),
				Name:           groupData["name"].(string),
				OrganizationID: uuid.MustParse(groupData["organization_id"].(string)),
			}
			if groupData["description"] != nil {
				desc := groupData["description"].(string)
				group.Description = sql.NullString{String: desc, Valid: true}
			}
			group.CreatedAt = time.Now()
			group.UpdatedAt = time.Now()

			suite.mockRepo.On("GetSecretGroupByID", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return(group, nil).Once()
		}
	case "GetSecretGroupByName":
		if config.Return["error"] != nil {
			errorMsg := config.Return["error"].(string)
			if errorMsg == "sql: no rows in result set" {
				suite.mockRepo.On("GetSecretGroupByName", suite.ctx, mock.AnythingOfType("secretgroupdb.GetSecretGroupByNameParams")).
					Return(secretgroupdb.SecretGroup{}, sql.ErrNoRows).Once()
			} else {
				suite.mockRepo.On("GetSecretGroupByName", suite.ctx, mock.AnythingOfType("secretgroupdb.GetSecretGroupByNameParams")).
					Return(secretgroupdb.SecretGroup{}, errors.New(errorMsg)).Once()
			}
		} else {
			// Build mock secret group from test data
			groupData := config.Return["secret_group"].(map[string]interface{})
			group := secretgroupdb.SecretGroup{
				ID:             uuid.MustParse(groupData["id"].(string)),
				Name:           groupData["name"].(string),
				OrganizationID: uuid.MustParse(groupData["organization_id"].(string)),
			}
			if groupData["description"] != nil {
				desc := groupData["description"].(string)
				group.Description = sql.NullString{String: desc, Valid: true}
			}
			group.CreatedAt = time.Now()
			group.UpdatedAt = time.Now()

			suite.mockRepo.On("GetSecretGroupByName", suite.ctx, mock.AnythingOfType("secretgroupdb.GetSecretGroupByNameParams")).
				Return(group, nil).Once()
		}
	case "UpdateSecretGroup":
		if config.Return["error"] != nil {
			errorMsg := config.Return["error"].(string)
			if errorMsg == "sql: no rows in result set" {
				suite.mockRepo.On("UpdateSecretGroup", suite.ctx, mock.AnythingOfType("secretgroupdb.UpdateSecretGroupParams")).
					Return(secretgroupdb.SecretGroup{}, sql.ErrNoRows).Once()
			} else {
				suite.mockRepo.On("UpdateSecretGroup", suite.ctx, mock.AnythingOfType("secretgroupdb.UpdateSecretGroupParams")).
					Return(secretgroupdb.SecretGroup{}, errors.New(errorMsg)).Once()
			}
		} else {
			// Build mock secret group from test data
			groupData := config.Return["secret_group"].(map[string]interface{})
			group := secretgroupdb.SecretGroup{
				ID:             uuid.MustParse(groupData["id"].(string)),
				Name:           groupData["name"].(string),
				OrganizationID: uuid.MustParse(groupData["organization_id"].(string)),
			}
			if groupData["description"] != nil {
				desc := groupData["description"].(string)
				group.Description = sql.NullString{String: desc, Valid: true}
			}
			group.CreatedAt = time.Now()
			group.UpdatedAt = time.Now()

			suite.mockRepo.On("UpdateSecretGroup", suite.ctx, mock.AnythingOfType("secretgroupdb.UpdateSecretGroupParams")).
				Return(group, nil).Once()
		}
	case "DeleteSecretGroup":
		if config.Return["error"] != nil {
			errorMsg := config.Return["error"].(string)
			if errorMsg == "sql: no rows in result set" {
				suite.mockRepo.On("DeleteSecretGroup", suite.ctx, mock.AnythingOfType("uuid.UUID")).
					Return(sql.ErrNoRows).Once()
			} else if errorMsg == "foreign key constraint violation" {
				suite.mockRepo.On("DeleteSecretGroup", suite.ctx, mock.AnythingOfType("uuid.UUID")).
					Return(errors.New("foreign key constraint violation")).Once()
			} else {
				suite.mockRepo.On("DeleteSecretGroup", suite.ctx, mock.AnythingOfType("uuid.UUID")).
					Return(errors.New(errorMsg)).Once()
			}
		} else {
			suite.mockRepo.On("DeleteSecretGroup", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return(nil).Once()
		}
	}
}

// setupIamServiceMock sets up the IAM service mock
func (suite *SecretGroupServiceTestSuite) setupIamServiceMock(config MockConfig) {
	switch config.Method {
	case "CreateRoleBinding":
		if config.Return["error"] != nil {
			suite.mockIamRepo.On("CreateRoleBinding", suite.ctx, mock.AnythingOfType("iam_db.CreateRoleBindingParams")).
				Return(iam_db.RoleBinding{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock role binding from test data
			bindingData := config.Return["role_binding"].(map[string]interface{})
			binding := iam_db.RoleBinding{
				ID:             uuid.MustParse(bindingData["id"].(string)),
				UserID:         uuid.NullUUID{UUID: uuid.MustParse(bindingData["user_id"].(string)), Valid: true},
				Role:           iam_db.UserRole(bindingData["role"].(string)),
				ResourceType:   iam_db.ResourceType(bindingData["resource_type"].(string)),
				ResourceID:     uuid.MustParse(bindingData["resource_id"].(string)),
				OrganizationID: uuid.MustParse(bindingData["organization_id"].(string)),
			}
			binding.CreatedAt = time.Now()
			binding.UpdatedAt = time.Now()

			suite.mockIamRepo.On("CreateRoleBinding", suite.ctx, mock.AnythingOfType("iam_db.CreateRoleBindingParams")).
				Return(binding, nil).Once()
		}
	case "ListAccessibleSecretGroups":
		if config.Return["error"] != nil {
			suite.mockIamRepo.On("ListAccessibleSecretGroups", suite.ctx, mock.AnythingOfType("iam_db.ListAccessibleSecretGroupsParams")).
				Return([]iam_db.ListAccessibleSecretGroupsRow{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock accessible secret groups from test data
			groups := []iam_db.ListAccessibleSecretGroupsRow{}
			if config.Return["secret_groups"] != nil {
				for _, group := range config.Return["secret_groups"].([]interface{}) {
					groupMap := group.(map[string]interface{})
					groupItem := iam_db.ListAccessibleSecretGroupsRow{
						ID:               uuid.NullUUID{UUID: uuid.MustParse(groupMap["id"].(string)), Valid: true},
						Name:             groupMap["name"].(string),
						OrganizationName: groupMap["organization_name"].(string),
						Role:             iam_db.UserRole(groupMap["role"].(string)),
					}
					groups = append(groups, groupItem)
				}
			}
			suite.mockIamRepo.On("ListAccessibleSecretGroups", suite.ctx, mock.AnythingOfType("iam_db.ListAccessibleSecretGroupsParams")).
				Return(groups, nil).Once()
		}
	case "DeleteRoleBinding":
		if config.Return["error"] != nil {
			suite.mockIamRepo.On("DeleteRoleBinding", suite.ctx, mock.AnythingOfType("iam_db.DeleteRoleBindingParams")).
				Return(errors.New(config.Return["error"].(string))).Once()
		} else {
			suite.mockIamRepo.On("DeleteRoleBinding", suite.ctx, mock.AnythingOfType("iam_db.DeleteRoleBindingParams")).
				Return(nil).Once()
		}
	}
}

// setupPolicyEnforcerMock sets up the policy enforcer mock
func (suite *SecretGroupServiceTestSuite) setupPolicyEnforcerMock(config MockConfig) {
	switch config.Method {
	case "AddResourceOwner":
		if config.Return["error"] != nil {
			suite.mockPolicyEnforcer.MockEnforcer.On("AddResourceOwner", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(errors.New(config.Return["error"].(string))).Once()
		} else {
			suite.mockPolicyEnforcer.MockEnforcer.On("AddResourceOwner", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(nil).Once()
		}
	case "AddResourceHierarchy":
		if config.Return["error"] != nil {
			suite.mockPolicyEnforcer.MockEnforcer.On("AddResourceHierarchy", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(errors.New(config.Return["error"].(string))).Once()
		} else {
			suite.mockPolicyEnforcer.MockEnforcer.On("AddResourceHierarchy", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(nil).Once()
		}
	case "RemoveResource":
		if config.Return["error"] != nil {
			suite.mockPolicyEnforcer.MockEnforcer.On("RemoveResource", mock.AnythingOfType("string")).
				Return(errors.New(config.Return["error"].(string))).Once()
		} else {
			suite.mockPolicyEnforcer.MockEnforcer.On("RemoveResource", mock.AnythingOfType("string")).
				Return(nil).Once()
		}
	case "RemoveResourceHierarchy":
		if config.Return["error"] != nil {
			suite.mockPolicyEnforcer.MockEnforcer.On("RemoveResourceHierarchy", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(errors.New(config.Return["error"].(string))).Once()
		} else {
			suite.mockPolicyEnforcer.MockEnforcer.On("RemoveResourceHierarchy", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(nil).Once()
		}
	}
}

// buildCreateSecretGroupRequest builds a CreateSecretGroupRequest from test input
func (suite *SecretGroupServiceTestSuite) buildCreateSecretGroupRequest(input map[string]interface{}) CreateSecretGroupRequest {
	req := CreateSecretGroupRequest{
		Name:           input["name"].(string),
		UserID:         input["user_id"].(string),
		OrganizationID: input["organization_id"].(string),
	}
	if input["description"] != nil {
		req.Description = input["description"].(string)
	}
	if input["organization_name"] != nil {
		req.OrganizationName = input["organization_name"].(string)
	}
	return req
}

// buildUpdateSecretGroupRequest builds an UpdateSecretGroupRequest from test input
func (suite *SecretGroupServiceTestSuite) buildUpdateSecretGroupRequest(input map[string]interface{}) UpdateSecretGroupRequest {
	req := UpdateSecretGroupRequest{
		Name: input["name"].(string),
	}
	if input["description"] != nil {
		req.Description = input["description"].(string)
	}
	return req
}

// TestSecretGroupServiceTestSuite runs the test suite
func TestSecretGroupServiceTestSuite(t *testing.T) {
	suite.Run(t, new(SecretGroupServiceTestSuite))
}
