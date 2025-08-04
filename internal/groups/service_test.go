package groups

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
	appErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	groupsdb "github.com/Gkemhcs/kavach-backend/internal/groups/gen"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//go:embed test_data/*.json
var testDataFS embed.FS

// TestData represents the structure of our test data files
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
	Success   bool        `json:"success"`
	Error     interface{} `json:"error"`
	ErrorCode string      `json:"error_code,omitempty"`
	// Additional fields for specific test types
	UserGroup  interface{}              `json:"user_group,omitempty"`
	UserGroups []map[string]interface{} `json:"user_groups,omitempty"`
	Members    []map[string]interface{} `json:"members,omitempty"`
}

// MockSetup represents the mock configuration for a test case
type MockSetup struct {
	UserGroupRepo  MockConfig   `json:"user_group_repo,omitempty"`
	UserService    MockConfig   `json:"user_service,omitempty"`
	PolicyEnforcer []MockConfig `json:"policy_enforcer,omitempty"`
}

// MockConfig represents a single mock configuration
type MockConfig struct {
	Method string                 `json:"method"`
	Return map[string]interface{} `json:"return"`
}

// UserGroupServiceTestSuite represents the test suite for UserGroupService
type UserGroupServiceTestSuite struct {
	suite.Suite
	service            *UserGroupService
	mockRepo           *MockUserGroupRepository
	mockUserService    *MockUserService
	mockPolicyEnforcer *MockPolicyEnforcer
	logger             *logrus.Logger
	ctx                context.Context
}

// MockUserGroupRepository is a mock implementation of the groupsdb.Querier interface
type MockUserGroupRepository struct {
	mock.Mock
}

// MockUserService is a mock implementation of the auth.UserInfoGetter interface
type MockUserService struct {
	mock.Mock
}

// MockPolicyEnforcer is a mock implementation of the authz.Enforcer interface
type MockPolicyEnforcer struct {
	*authz.MockEnforcer
}

// SetupSuite sets up the test suite
func (suite *UserGroupServiceTestSuite) SetupSuite() {
	// Suppress logrus output during tests
	suite.logger = logrus.New()
	suite.logger.SetOutput(io.Discard)
}

// SetupTest sets up each individual test
func (suite *UserGroupServiceTestSuite) SetupTest() {
	suite.mockRepo = &MockUserGroupRepository{}
	suite.mockUserService = &MockUserService{}
	suite.mockPolicyEnforcer = &MockPolicyEnforcer{MockEnforcer: &authz.MockEnforcer{}}

	suite.service = NewUserGroupService(
		suite.logger,
		suite.mockRepo,
		suite.mockUserService,
		suite.mockPolicyEnforcer,
	)

	suite.ctx = context.Background()
}

// TearDownTest cleans up after each test
func (suite *UserGroupServiceTestSuite) TearDownTest() {
	// Reset all mock expectations
	suite.mockRepo.ExpectedCalls = nil
	suite.mockUserService.ExpectedCalls = nil
	suite.mockPolicyEnforcer.ExpectedCalls = nil
}

// loadTestData loads test data from embedded JSON files
func (suite *UserGroupServiceTestSuite) loadTestData(filename string) *TestData {
	data, err := testDataFS.ReadFile("test_data/" + filename)
	require.NoError(suite.T(), err, "Failed to read test data file: %s", filename)

	var testData TestData
	err = json.Unmarshal(data, &testData)
	require.NoError(suite.T(), err, "Failed to unmarshal test data from: %s", filename)

	return &testData
}

// validateErrorCode validates that the error has the expected error code or message
func (suite *UserGroupServiceTestSuite) validateErrorCode(err error, expectedErrorCode string) {
	if expectedErrorCode == "" {
		return
	}

	// Try to cast to APIError to get the code
	if apiErr, ok := err.(*appErrors.APIError); ok {
		require.Equal(suite.T(), expectedErrorCode, apiErr.Code, "Error code mismatch")
	} else {
		// Fallback to checking if the error message contains the expected text
		// This is for cases where the error might be wrapped or is a plain error
		require.Contains(suite.T(), err.Error(), expectedErrorCode, "Error message does not contain expected text")
	}
}

// createPostgresError creates a PostgreSQL-style error for testing
func (suite *UserGroupServiceTestSuite) createPostgresError(code, message string) error {
	// Note: This is a simplified approach for testing
	// In a real scenario, you might want to create actual pq.Error instances

	// For now, let's create a custom error that can be detected by the error functions
	return &postgresTestError{
		code:    code,
		message: message,
	}
}

// postgresTestError is a test-only error type that simulates PostgreSQL errors
type postgresTestError struct {
	code    string
	message string
}

func (e *postgresTestError) Error() string {
	return e.message
}

func (e *postgresTestError) Code() string {
	return e.code
}

// TestCreateUserGroupWithData tests CreateUserGroup with data-driven test cases
func (suite *UserGroupServiceTestSuite) TestCreateUserGroupWithData() {
	testData := suite.loadTestData("create_user_group_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupCreateUserGroupMocks(tc.MockSetup)

			// Build request from test input
			req := suite.buildCreateUserGroupRequest(tc.Input)

			// Call the service method
			result, err := suite.service.CreateUserGroup(suite.ctx, req)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedGroup := tc.Expected.UserGroup.(map[string]interface{})
				assert.Equal(suite.T(), expectedGroup["name"].(string), result.Name, "User group name mismatch")
				assert.Equal(suite.T(), expectedGroup["organization_id"].(string), result.OrganizationID.String(), "Organization ID mismatch")
				if expectedGroup["description"] != nil && result.Description.Valid {
					assert.Equal(suite.T(), expectedGroup["description"].(string), result.Description.String, "Description mismatch")
				}
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				// Validate error code or message if specified
				if tc.Expected.ErrorCode != "" {
					suite.validateErrorCode(err, tc.Expected.ErrorCode)
				} else if tc.Expected.Error != nil {
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

// TestGetUserGroupByNameWithData tests GetUserGroupByName with data-driven test cases
func (suite *UserGroupServiceTestSuite) TestGetUserGroupByNameWithData() {
	testData := suite.loadTestData("get_user_group_by_name_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupGetUserGroupByNameMocks(tc.MockSetup)

			// Get input parameters
			groupName := tc.Input["group_name"].(string)
			orgID := tc.Input["org_id"].(string)

			// Call the service method
			result, err := suite.service.GetUserGroupByName(suite.ctx, groupName, orgID)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedGroup := tc.Expected.UserGroup.(map[string]interface{})
				assert.Equal(suite.T(), expectedGroup["name"].(string), result.Name, "User group name mismatch")
				assert.Equal(suite.T(), expectedGroup["organization_id"].(string), result.OrganizationID.String(), "Organization ID mismatch")
				if expectedGroup["description"] != nil && result.Description.Valid {
					assert.Equal(suite.T(), expectedGroup["description"].(string), result.Description.String, "Description mismatch")
				}
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				// Validate error code or message if specified
				if tc.Expected.ErrorCode != "" {
					suite.validateErrorCode(err, tc.Expected.ErrorCode)
				} else if tc.Expected.Error != nil {
					expectedError := strings.ToLower(fmt.Sprintf("%v", tc.Expected.Error))
					actualError := strings.ToLower(err.Error())
					require.Contains(suite.T(), actualError, expectedError, "Error message mismatch")
				}
			}
		})
	}
}

// TestDeleteUserGroupWithData tests DeleteUserGroup with data-driven test cases
func (suite *UserGroupServiceTestSuite) TestDeleteUserGroupWithData() {
	testData := suite.loadTestData("delete_user_group_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupDeleteUserGroupMocks(tc.MockSetup)

			// Get input parameters
			orgID := tc.Input["org_id"].(string)
			groupID := tc.Input["group_id"].(string)

			// Call the service method
			err := suite.service.DeleteUserGroup(suite.ctx, orgID, groupID)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				// Validate error code or message if specified
				if tc.Expected.ErrorCode != "" {
					suite.validateErrorCode(err, tc.Expected.ErrorCode)
				} else if tc.Expected.Error != nil {
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

// TestListUserGroupsWithData tests ListUserGroups with data-driven test cases
func (suite *UserGroupServiceTestSuite) TestListUserGroupsWithData() {
	testData := suite.loadTestData("list_user_groups_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupListUserGroupsMocks(tc.MockSetup)

			// Get input parameters
			orgID := tc.Input["org_id"].(string)

			// Call the service method
			result, err := suite.service.ListUserGroups(suite.ctx, orgID)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedGroups := tc.Expected.UserGroups
				assert.Equal(suite.T(), len(expectedGroups), len(result), "Number of user groups mismatch")

				for i, expectedGroup := range expectedGroups {
					if i < len(result) {
						assert.Equal(suite.T(), expectedGroup["name"].(string), result[i].Name, "User group name mismatch")
						if expectedGroup["description"] != nil && result[i].Description.Valid {
							assert.Equal(suite.T(), expectedGroup["description"].(string), result[i].Description.String, "Description mismatch")
						}
					}
				}
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				// Validate error code or message if specified
				if tc.Expected.ErrorCode != "" {
					suite.validateErrorCode(err, tc.Expected.ErrorCode)
				} else if tc.Expected.Error != nil {
					expectedError := strings.ToLower(fmt.Sprintf("%v", tc.Expected.Error))
					actualError := strings.ToLower(err.Error())
					require.Contains(suite.T(), actualError, expectedError, "Error message mismatch")
				}
			}
		})
	}
}

// TestAddGroupMemberWithData tests AddGroupMember with data-driven test cases
func (suite *UserGroupServiceTestSuite) TestAddGroupMemberWithData() {
	testData := suite.loadTestData("add_group_member_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupAddGroupMemberMocks(tc.MockSetup)

			// Build request from test input
			req := suite.buildAddGroupMemberRequest(tc.Input)

			// Call the service method
			err := suite.service.AddGroupMember(suite.ctx, req)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				// Validate error code or message if specified
				if tc.Expected.ErrorCode != "" {
					suite.validateErrorCode(err, tc.Expected.ErrorCode)
				} else if tc.Expected.Error != nil {
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

// TestRemoveGroupMemberWithData tests RemoveGroupMember with data-driven test cases
func (suite *UserGroupServiceTestSuite) TestRemoveGroupMemberWithData() {
	testData := suite.loadTestData("remove_group_member_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupRemoveGroupMemberMocks(tc.MockSetup)

			// Build request from test input
			req := suite.buildRemoveGroupMemberRequest(tc.Input)

			// Call the service method
			err := suite.service.RemoveGroupMember(suite.ctx, req)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				// Validate error code or message if specified
				if tc.Expected.ErrorCode != "" {
					suite.validateErrorCode(err, tc.Expected.ErrorCode)
				} else if tc.Expected.Error != nil {
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

// TestListGroupMembersWithData tests ListGroupMembers with data-driven test cases
func (suite *UserGroupServiceTestSuite) TestListGroupMembersWithData() {
	testData := suite.loadTestData("list_group_members_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupListGroupMembersMocks(tc.MockSetup)

			// Get input parameters
			groupID := tc.Input["group_id"].(string)

			// Call the service method
			result, err := suite.service.ListGroupMembers(suite.ctx, groupID)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedMembers := tc.Expected.Members
				assert.Equal(suite.T(), len(expectedMembers), len(result), "Number of members mismatch")

				for i, expectedMember := range expectedMembers {
					if i < len(result) {
						if expectedMember["name"] != nil && result[i].Name.Valid {
							assert.Equal(suite.T(), expectedMember["name"].(string), result[i].Name.String, "Member name mismatch")
						}
						if expectedMember["email"] != nil && result[i].Email.Valid {
							assert.Equal(suite.T(), expectedMember["email"].(string), result[i].Email.String, "Member email mismatch")
						}
					}
				}
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				// Validate error code or message if specified
				if tc.Expected.ErrorCode != "" {
					suite.validateErrorCode(err, tc.Expected.ErrorCode)
				} else if tc.Expected.Error != nil {
					expectedError := strings.ToLower(fmt.Sprintf("%v", tc.Expected.Error))
					actualError := strings.ToLower(err.Error())
					require.Contains(suite.T(), actualError, expectedError, "Error message mismatch")
				}
			}
		})
	}
}

// Mock implementations for the repository interface
func (m *MockUserGroupRepository) CreateGroup(ctx context.Context, arg groupsdb.CreateGroupParams) (groupsdb.UserGroup, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return groupsdb.UserGroup{}, args.Error(1)
	}
	return args.Get(0).(groupsdb.UserGroup), args.Error(1)
}

func (m *MockUserGroupRepository) GetGroupByName(ctx context.Context, arg groupsdb.GetGroupByNameParams) (groupsdb.UserGroup, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return groupsdb.UserGroup{}, args.Error(1)
	}
	return args.Get(0).(groupsdb.UserGroup), args.Error(1)
}

func (m *MockUserGroupRepository) DeleteGroup(ctx context.Context, arg groupsdb.DeleteGroupParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockUserGroupRepository) ListGroupsByOrg(ctx context.Context, organizationID uuid.UUID) ([]groupsdb.ListGroupsByOrgRow, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]groupsdb.ListGroupsByOrgRow), args.Error(1)
}

func (m *MockUserGroupRepository) AddGroupMember(ctx context.Context, arg groupsdb.AddGroupMemberParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockUserGroupRepository) RemoveGroupMember(ctx context.Context, arg groupsdb.RemoveGroupMemberParams) (sql.Result, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(sql.Result), args.Error(1)
}

func (m *MockUserGroupRepository) ListGroupMembers(ctx context.Context, userGroupID uuid.UUID) ([]groupsdb.ListGroupMembersRow, error) {
	args := m.Called(ctx, userGroupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]groupsdb.ListGroupMembersRow), args.Error(1)
}

// Mock implementations for the user service interface
func (m *MockUserService) GetUserInfoByGithubUserName(ctx context.Context, username string) (*userdb.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userdb.User), args.Error(1)
}

// Setup functions for different test methods
func (suite *UserGroupServiceTestSuite) setupCreateUserGroupMocks(mockSetup MockSetup) {
	if mockSetup.UserGroupRepo.Method != "" {
		suite.setupUserGroupRepoMock(mockSetup.UserGroupRepo)
	}

	for _, policyConfig := range mockSetup.PolicyEnforcer {
		if policyConfig.Method != "" {
			suite.setupPolicyEnforcerMock(policyConfig)
		}
	}
}

func (suite *UserGroupServiceTestSuite) setupGetUserGroupByNameMocks(mockSetup MockSetup) {
	if mockSetup.UserGroupRepo.Method != "" {
		suite.setupUserGroupRepoMock(mockSetup.UserGroupRepo)
	}
}

func (suite *UserGroupServiceTestSuite) setupDeleteUserGroupMocks(mockSetup MockSetup) {
	if mockSetup.UserGroupRepo.Method != "" {
		suite.setupUserGroupRepoMock(mockSetup.UserGroupRepo)
	}

	for _, policyConfig := range mockSetup.PolicyEnforcer {
		if policyConfig.Method != "" {
			suite.setupPolicyEnforcerMock(policyConfig)
		}
	}
}

func (suite *UserGroupServiceTestSuite) setupListUserGroupsMocks(mockSetup MockSetup) {
	if mockSetup.UserGroupRepo.Method != "" {
		suite.setupUserGroupRepoMock(mockSetup.UserGroupRepo)
	}
}

func (suite *UserGroupServiceTestSuite) setupAddGroupMemberMocks(mockSetup MockSetup) {
	if mockSetup.UserService.Method != "" {
		suite.setupUserServiceMock(mockSetup.UserService)
	}

	if mockSetup.UserGroupRepo.Method != "" {
		suite.setupUserGroupRepoMock(mockSetup.UserGroupRepo)
	}

	for _, policyConfig := range mockSetup.PolicyEnforcer {
		if policyConfig.Method != "" {
			suite.setupPolicyEnforcerMock(policyConfig)
		}
	}
}

func (suite *UserGroupServiceTestSuite) setupRemoveGroupMemberMocks(mockSetup MockSetup) {
	if mockSetup.UserService.Method != "" {
		suite.setupUserServiceMock(mockSetup.UserService)
	}

	if mockSetup.UserGroupRepo.Method != "" {
		suite.setupUserGroupRepoMock(mockSetup.UserGroupRepo)
	}

	for _, policyConfig := range mockSetup.PolicyEnforcer {
		if policyConfig.Method != "" {
			suite.setupPolicyEnforcerMock(policyConfig)
		}
	}
}

func (suite *UserGroupServiceTestSuite) setupListGroupMembersMocks(mockSetup MockSetup) {
	if mockSetup.UserGroupRepo.Method != "" {
		suite.setupUserGroupRepoMock(mockSetup.UserGroupRepo)
	}
}

// validateNegativeMocks validates that certain methods were NOT called when they shouldn't be
func (suite *UserGroupServiceTestSuite) validateNegativeMocks(testCase string) {
	// CreateUserGroup error cases - these should NOT call downstream methods
	if strings.Contains(testCase, "database_error") {
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "AddResourceOwner")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "AddResourceHierarchy")
	}

	// DeleteUserGroup error cases
	if strings.Contains(testCase, "group_not_found") {
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "DeleteUserGroup")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "RemoveResourceHierarchy")
	}

	// AddGroupMember error cases
	if strings.Contains(testCase, "user_not_found") {
		suite.mockRepo.AssertNotCalled(suite.T(), "AddGroupMember")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "AddUserToGroup")
	}

	if strings.Contains(testCase, "database_error") && strings.Contains(testCase, "add_member") {
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "AddUserToGroup")
	}

	// RemoveGroupMember error cases
	if strings.Contains(testCase, "user_not_found") && strings.Contains(testCase, "remove_member") {
		suite.mockRepo.AssertNotCalled(suite.T(), "RemoveGroupMember")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "RemoveUserFromGroup")
	}

	if strings.Contains(testCase, "database_error") && strings.Contains(testCase, "remove_member") {
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "RemoveUserFromGroup")
	}
}

// setupUserGroupRepoMock sets up the user group repository mock
func (suite *UserGroupServiceTestSuite) setupUserGroupRepoMock(config MockConfig) {
	switch config.Method {
	case "CreateGroup":
		if config.Return["error"] != nil {
			errorMsg := config.Return["error"].(string)
			var err error

			// Handle specific error types
			switch errorMsg {
			case "duplicate key value violates unique constraint":
				// Create a PostgreSQL-style error for unique constraint violation
				err = &postgresTestError{code: "23505", message: errorMsg}
			case "sql: no rows in result set":
				err = sql.ErrNoRows
			default:
				err = errors.New(errorMsg)
			}

			suite.mockRepo.On("CreateGroup", suite.ctx, mock.AnythingOfType("groupsdb.CreateGroupParams")).
				Return(groupsdb.UserGroup{}, err).Once()
		} else {
			// Build mock user group from test data
			groupData := config.Return["user_group"].(map[string]interface{})
			group := groupsdb.UserGroup{
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

			suite.mockRepo.On("CreateGroup", suite.ctx, mock.AnythingOfType("groupsdb.CreateGroupParams")).
				Return(group, nil).Once()
		}
	case "GetGroupByName":
		if config.Return["error"] != nil {
			errorMsg := config.Return["error"].(string)
			var err error

			// Handle specific error types
			switch errorMsg {
			case "sql: no rows in result set":
				err = sql.ErrNoRows
			default:
				err = errors.New(errorMsg)
			}

			suite.mockRepo.On("GetGroupByName", suite.ctx, mock.AnythingOfType("groupsdb.GetGroupByNameParams")).
				Return(groupsdb.UserGroup{}, err).Once()
		} else {
			// Build mock user group from test data
			groupData := config.Return["user_group"].(map[string]interface{})
			group := groupsdb.UserGroup{
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

			suite.mockRepo.On("GetGroupByName", suite.ctx, mock.AnythingOfType("groupsdb.GetGroupByNameParams")).
				Return(group, nil).Once()
		}
	case "DeleteGroup":
		if config.Return["error"] != nil {
			errorMsg := config.Return["error"].(string)
			var err error

			// Handle specific error types
			switch errorMsg {
			case "sql: no rows in result set":
				err = sql.ErrNoRows
			default:
				err = errors.New(errorMsg)
			}

			suite.mockRepo.On("DeleteGroup", suite.ctx, mock.AnythingOfType("groupsdb.DeleteGroupParams")).
				Return(err).Once()
		} else {
			suite.mockRepo.On("DeleteGroup", suite.ctx, mock.AnythingOfType("groupsdb.DeleteGroupParams")).
				Return(nil).Once()
		}
	case "ListGroupsByOrg":
		if config.Return["error"] != nil {
			suite.mockRepo.On("ListGroupsByOrg", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return([]groupsdb.ListGroupsByOrgRow{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock user groups from test data
			groups := []groupsdb.ListGroupsByOrgRow{}
			if config.Return["user_groups"] != nil {
				for _, group := range config.Return["user_groups"].([]interface{}) {
					groupMap := group.(map[string]interface{})
					groupItem := groupsdb.ListGroupsByOrgRow{
						ID:   uuid.MustParse(groupMap["id"].(string)),
						Name: groupMap["name"].(string),
					}
					if groupMap["description"] != nil {
						desc := groupMap["description"].(string)
						groupItem.Description = sql.NullString{String: desc, Valid: true}
					}
					groupItem.CreatedAt = time.Now()
					groups = append(groups, groupItem)
				}
			}
			suite.mockRepo.On("ListGroupsByOrg", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return(groups, nil).Once()
		}
	case "AddGroupMember":
		if config.Return["error"] != nil {
			errorMsg := config.Return["error"].(string)
			var err error

			// Handle specific error types
			switch errorMsg {
			case "duplicate key value violates unique constraint":
				// Create a PostgreSQL-style error for unique constraint violation
				err = &postgresTestError{code: "23505", message: errorMsg}
			default:
				err = errors.New(errorMsg)
			}

			suite.mockRepo.On("AddGroupMember", suite.ctx, mock.AnythingOfType("groupsdb.AddGroupMemberParams")).
				Return(err).Once()
		} else {
			suite.mockRepo.On("AddGroupMember", suite.ctx, mock.AnythingOfType("groupsdb.AddGroupMemberParams")).
				Return(nil).Once()
		}
	case "RemoveGroupMember":
		if config.Return["error"] != nil {
			suite.mockRepo.On("RemoveGroupMember", suite.ctx, mock.AnythingOfType("groupsdb.RemoveGroupMemberParams")).
				Return(nil, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Create a mock sql.Result
			mockResult := &mockSqlResult{}
			if config.Return["rows_affected"] != nil {
				mockResult.rowsAffected = int64(config.Return["rows_affected"].(float64))
			}
			suite.mockRepo.On("RemoveGroupMember", suite.ctx, mock.AnythingOfType("groupsdb.RemoveGroupMemberParams")).
				Return(mockResult, nil).Once()
		}
	case "ListGroupMembers":
		if config.Return["error"] != nil {
			suite.mockRepo.On("ListGroupMembers", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return([]groupsdb.ListGroupMembersRow{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock members from test data
			members := []groupsdb.ListGroupMembersRow{}
			if config.Return["members"] != nil {
				for _, member := range config.Return["members"].([]interface{}) {
					memberMap := member.(map[string]interface{})
					memberItem := groupsdb.ListGroupMembersRow{
						ID: uuid.MustParse(memberMap["id"].(string)),
					}
					if memberMap["name"] != nil {
						name := memberMap["name"].(string)
						memberItem.Name = sql.NullString{String: name, Valid: true}
					}
					if memberMap["email"] != nil {
						email := memberMap["email"].(string)
						memberItem.Email = sql.NullString{String: email, Valid: true}
					}
					memberItem.CreatedAt = time.Now()
					members = append(members, memberItem)
				}
			}
			suite.mockRepo.On("ListGroupMembers", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return(members, nil).Once()
		}
	}
}

// setupUserServiceMock sets up the user service mock
func (suite *UserGroupServiceTestSuite) setupUserServiceMock(config MockConfig) {
	switch config.Method {
	case "GetUserInfoByGithubUserName":
		if config.Return["error"] != nil {
			errorMsg := config.Return["error"].(string)
			var err error

			// Handle specific error types
			switch errorMsg {
			case "sql: no rows in result set":
				err = sql.ErrNoRows
			default:
				err = errors.New(errorMsg)
			}

			suite.mockUserService.On("GetUserInfoByGithubUserName", suite.ctx, mock.AnythingOfType("string")).
				Return(nil, err).Once()
		} else {
			// Build mock user from test data
			userData := config.Return["user"].(map[string]interface{})
			user := &userdb.User{
				ID: uuid.MustParse(userData["id"].(string)),
			}
			if userData["name"] != nil {
				name := userData["name"].(string)
				user.Name = sql.NullString{String: name, Valid: true}
			}
			if userData["email"] != nil {
				email := userData["email"].(string)
				user.Email = sql.NullString{String: email, Valid: true}
			}

			suite.mockUserService.On("GetUserInfoByGithubUserName", suite.ctx, mock.AnythingOfType("string")).
				Return(user, nil).Once()
		}
	}
}

// setupPolicyEnforcerMock sets up the policy enforcer mock
func (suite *UserGroupServiceTestSuite) setupPolicyEnforcerMock(config MockConfig) {
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
	case "DeleteUserGroup":
		if config.Return["error"] != nil {
			suite.mockPolicyEnforcer.MockEnforcer.On("DeleteUserGroup", mock.AnythingOfType("string")).
				Return(errors.New(config.Return["error"].(string))).Once()
		} else {
			suite.mockPolicyEnforcer.MockEnforcer.On("DeleteUserGroup", mock.AnythingOfType("string")).
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
	case "AddUserToGroup":
		if config.Return["error"] != nil {
			suite.mockPolicyEnforcer.MockEnforcer.On("AddUserToGroup", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(errors.New(config.Return["error"].(string))).Once()
		} else {
			suite.mockPolicyEnforcer.MockEnforcer.On("AddUserToGroup", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(nil).Once()
		}
	case "RemoveUserFromGroup":
		if config.Return["error"] != nil {
			suite.mockPolicyEnforcer.MockEnforcer.On("RemoveUserFromGroup", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(errors.New(config.Return["error"].(string))).Once()
		} else {
			suite.mockPolicyEnforcer.MockEnforcer.On("RemoveUserFromGroup", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(nil).Once()
		}
	}
}

// Helper functions to build requests from test input
func (suite *UserGroupServiceTestSuite) buildCreateUserGroupRequest(input map[string]interface{}) CreateUserGroupRequest {
	req := CreateUserGroupRequest{
		GroupName:      input["group_name"].(string),
		OrganizationID: input["organization_id"].(string),
	}
	if input["description"] != nil {
		req.Description = input["description"].(string)
	}
	if input["user_id"] != nil {
		req.UserID = input["user_id"].(string)
	}
	return req
}

func (suite *UserGroupServiceTestSuite) buildAddGroupMemberRequest(input map[string]interface{}) AddMemberRequest {
	req := AddMemberRequest{
		UserGroupID: input["user_group_id"].(string),
		UserName:    input["user_name"].(string),
	}
	return req
}

func (suite *UserGroupServiceTestSuite) buildRemoveGroupMemberRequest(input map[string]interface{}) RemoveMemberRequest {
	req := RemoveMemberRequest{
		UserGroupID: input["user_group_id"].(string),
		UserName:    input["user_name"].(string),
	}
	return req
}

// mockSqlResult is a mock implementation of sql.Result for testing
type mockSqlResult struct {
	rowsAffected int64
}

func (m *mockSqlResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (m *mockSqlResult) RowsAffected() (int64, error) {
	return m.rowsAffected, nil
}

// TestUserGroupServiceTestSuite runs the test suite
func TestUserGroupServiceTestSuite(t *testing.T) {
	suite.Run(t, new(UserGroupServiceTestSuite))
}
