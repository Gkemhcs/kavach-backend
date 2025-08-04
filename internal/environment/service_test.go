package environment

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
	environmentdb "github.com/Gkemhcs/kavach-backend/internal/environment/gen"
	appErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	groupsdb "github.com/Gkemhcs/kavach-backend/internal/groups/gen"
	"github.com/Gkemhcs/kavach-backend/internal/iam"
	iam_db "github.com/Gkemhcs/kavach-backend/internal/iam/gen"
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
	Success   bool        `json:"success"`
	Error     interface{} `json:"error"`
	ErrorCode string      `json:"error_code,omitempty"`
	// Additional fields for specific test types
	Environment  interface{}              `json:"environment,omitempty"`
	Environments []map[string]interface{} `json:"environments,omitempty"`
}

// MockSetup represents the mock configuration for a test case
type MockSetup struct {
	EnvironmentRepo MockConfig   `json:"environment_repo,omitempty"`
	IamService      MockConfig   `json:"iam_service,omitempty"`
	PolicyEnforcer  []MockConfig `json:"policy_enforcer,omitempty"`
}

// MockConfig represents configuration for a specific mock
type MockConfig struct {
	Method string                 `json:"method"`
	Return map[string]interface{} `json:"return"`
}

// EnvironmentServiceTestSuite is the test suite for EnvironmentService
type EnvironmentServiceTestSuite struct {
	suite.Suite
	service            *EnvironmentService
	mockRepo           *MockEnvironmentRepository
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
func (suite *EnvironmentServiceTestSuite) SetupSuite() {
	suite.logger = logrus.New()
	suite.logger.SetOutput(io.Discard) // Disable logging to stdout
	suite.ctx = context.Background()
}

// SetupTest initializes each test
func (suite *EnvironmentServiceTestSuite) SetupTest() {
	suite.mockRepo = &MockEnvironmentRepository{}
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
	suite.service = NewEnvironmentService(
		suite.mockRepo,
		suite.logger,
		*suite.iamService,
		suite.mockPolicyEnforcer,
	)
}

// TearDownTest cleans up after each test
func (suite *EnvironmentServiceTestSuite) TearDownTest() {
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
func (suite *EnvironmentServiceTestSuite) loadTestData(filename string) *TestData {
	data, err := testDataFS.ReadFile("test_data/" + filename)
	require.NoError(suite.T(), err, "Failed to read test data file: %s", filename)

	var testData TestData
	err = json.Unmarshal(data, &testData)
	require.NoError(suite.T(), err, "Failed to unmarshal test data from: %s", filename)

	return &testData
}

// validateErrorCode validates that the error has the expected error code or message
func (suite *EnvironmentServiceTestSuite) validateErrorCode(err error, expectedErrorCode string) {
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
func (suite *EnvironmentServiceTestSuite) createPostgresError(code, message string) error {
	// Import pq package for PostgreSQL errors
	// Note: This is a simplified approach for testing
	// In a real scenario, you might want to create actual pq.Error instances

	// For now, let's create a custom error that can be detected by the error functions
	return &postgresTestError{
		code:    code,
		message: message,
	}
}

// postgresTestError is a test implementation of PostgreSQL error
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

// TestCreateEnvironmentWithData tests CreateEnvironment with data-driven test cases
func (suite *EnvironmentServiceTestSuite) TestCreateEnvironmentWithData() {
	testData := suite.loadTestData("create_environment_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupCreateEnvironmentMocks(tc.MockSetup)

			// Build request from test data
			req := suite.buildCreateEnvironmentRequest(tc.Input)

			// Call the service method
			result, err := suite.service.CreateEnvironment(suite.ctx, req)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedEnv := tc.Expected.Environment.(map[string]interface{})
				assert.Equal(suite.T(), expectedEnv["name"].(string), result.Name, "Environment name mismatch")
				assert.Equal(suite.T(), expectedEnv["secret_group_id"].(string), result.SecretGroupID.String(), "Secret group ID mismatch")
				if expectedEnv["description"] != nil && result.Description.Valid {
					assert.Equal(suite.T(), expectedEnv["description"].(string), result.Description.String, "Environment description mismatch")
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

// TestListEnvironmentsWithData tests ListEnvironments with data-driven test cases
func (suite *EnvironmentServiceTestSuite) TestListEnvironmentsWithData() {
	testData := suite.loadTestData("list_environments_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupListEnvironmentsMocks(tc.MockSetup)

			// Get input parameters
			userID := tc.Input["user_id"].(string)
			orgID := tc.Input["org_id"].(string)
			groupID := tc.Input["group_id"].(string)

			// Call the service method
			result, err := suite.service.ListEnvironments(suite.ctx, userID, orgID, groupID)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedEnvs := tc.Expected.Environments
				assert.Equal(suite.T(), len(expectedEnvs), len(result), "Number of environments mismatch")

				for i, expectedEnv := range expectedEnvs {
					if i < len(result) {
						assert.Equal(suite.T(), expectedEnv["name"].(string), result[i].Name, "Environment name mismatch")
						assert.Equal(suite.T(), expectedEnv["secret_group_id"].(string), result[i].SecretGroupID.String(), "Secret group ID mismatch")
						if expectedEnv["description"] != nil && result[i].Description.Valid {
							assert.Equal(suite.T(), expectedEnv["description"].(string), result[i].Description.String, "Environment description mismatch")
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

// TestListMyEnvironmentsWithData tests ListMyEnvironments with data-driven test cases
func (suite *EnvironmentServiceTestSuite) TestListMyEnvironmentsWithData() {
	testData := suite.loadTestData("list_my_environments_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupListMyEnvironmentsMocks(tc.MockSetup)

			// Get input parameters
			userID := tc.Input["user_id"].(string)
			groupID := tc.Input["group_id"].(string)
			orgID := tc.Input["org_id"].(string)

			// Call the service method
			result, err := suite.service.ListMyEnvironments(suite.ctx, userID, groupID, orgID)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedEnvs := tc.Expected.Environments
				assert.Equal(suite.T(), len(expectedEnvs), len(result), "Number of environments mismatch")

				for i, expectedEnv := range expectedEnvs {
					if i < len(result) {
						assert.Equal(suite.T(), expectedEnv["name"].(string), result[i].Name, "Environment name mismatch")
						assert.Equal(suite.T(), expectedEnv["secret_group_name"].(string), result[i].SecretGroupName, "Secret group name mismatch")
						assert.Equal(suite.T(), expectedEnv["role"].(string), string(result[i].Role), "Role mismatch")
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

// TestGetEnvironmentWithData tests GetEnvironment with data-driven test cases
func (suite *EnvironmentServiceTestSuite) TestGetEnvironmentWithData() {
	testData := suite.loadTestData("get_environment_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupGetEnvironmentMocks(tc.MockSetup)

			// Get input parameters
			userID := tc.Input["user_id"].(string)
			orgID := tc.Input["org_id"].(string)
			groupID := tc.Input["group_id"].(string)
			envID := tc.Input["env_id"].(string)

			// Call the service method
			result, err := suite.service.GetEnvironment(suite.ctx, userID, orgID, groupID, envID)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedEnv := tc.Expected.Environment.(map[string]interface{})
				assert.Equal(suite.T(), expectedEnv["name"].(string), result.Name, "Environment name mismatch")
				assert.Equal(suite.T(), expectedEnv["secret_group_id"].(string), result.SecretGroupID.String(), "Secret group ID mismatch")
				if expectedEnv["description"] != nil && result.Description.Valid {
					assert.Equal(suite.T(), expectedEnv["description"].(string), result.Description.String, "Environment description mismatch")
				}
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				// Validate error code if specified
				suite.validateErrorCode(err, tc.Expected.ErrorCode)
			}
		})
	}
}

// TestGetEnvironmentByNameWithData tests GetEnvironmentByName with data-driven test cases
func (suite *EnvironmentServiceTestSuite) TestGetEnvironmentByNameWithData() {
	testData := suite.loadTestData("get_environment_by_name_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupGetEnvironmentByNameMocks(tc.MockSetup)

			// Get input parameters
			environmentName := tc.Input["environment_name"].(string)
			groupID := tc.Input["group_id"].(string)

			// Call the service method
			result, err := suite.service.GetEnvironmentByName(suite.ctx, environmentName, groupID)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedEnv := tc.Expected.Environment.(map[string]interface{})
				assert.Equal(suite.T(), expectedEnv["name"].(string), result.Name, "Environment name mismatch")
				assert.Equal(suite.T(), expectedEnv["secret_group_id"].(string), result.SecretGroupID.String(), "Secret group ID mismatch")
				if expectedEnv["description"] != nil && result.Description.Valid {
					assert.Equal(suite.T(), expectedEnv["description"].(string), result.Description.String, "Environment description mismatch")
				}
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				// Validate error code if specified
				suite.validateErrorCode(err, tc.Expected.ErrorCode)
			}
		})
	}
}

// TestUpdateEnvironmentWithData tests UpdateEnvironment with data-driven test cases
func (suite *EnvironmentServiceTestSuite) TestUpdateEnvironmentWithData() {
	testData := suite.loadTestData("update_environment_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupUpdateEnvironmentMocks(tc.MockSetup)

			// Get input parameters
			userID := tc.Input["user_id"].(string)
			orgID := tc.Input["org_id"].(string)
			groupID := tc.Input["group_id"].(string)
			envID := tc.Input["env_id"].(string)
			name := tc.Input["name"].(string)

			// Build request
			req := UpdateEnvironmentRequest{
				Name: name,
			}

			// Call the service method
			result, err := suite.service.UpdateEnvironment(suite.ctx, userID, orgID, groupID, envID, req)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected result but got nil")

				// Validate result matches expected
				expectedEnv := tc.Expected.Environment.(map[string]interface{})
				assert.Equal(suite.T(), expectedEnv["name"].(string), result.Name, "Environment name mismatch")
				assert.Equal(suite.T(), expectedEnv["secret_group_id"].(string), result.SecretGroupID.String(), "Secret group ID mismatch")
				if expectedEnv["description"] != nil && result.Description.Valid {
					assert.Equal(suite.T(), expectedEnv["description"].(string), result.Description.String, "Environment description mismatch")
				}
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				// Validate error code if specified
				suite.validateErrorCode(err, tc.Expected.ErrorCode)
			}
		})
	}
}

// TestDeleteEnvironmentWithData tests DeleteEnvironment with data-driven test cases
func (suite *EnvironmentServiceTestSuite) TestDeleteEnvironmentWithData() {
	testData := suite.loadTestData("delete_environment_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			// Setup mocks based on test case
			suite.setupDeleteEnvironmentMocks(tc.MockSetup)

			// Get input parameters
			userID := tc.Input["user_id"].(string)
			orgID := tc.Input["org_id"].(string)
			groupID := tc.Input["group_id"].(string)
			envID := tc.Input["env_id"].(string)

			// Call the service method
			err := suite.service.DeleteEnvironment(suite.ctx, userID, orgID, groupID, envID)

			// Assert results
			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
			} else {
				require.Error(suite.T(), err, "Expected error but got success")
				// Validate error code if specified
				suite.validateErrorCode(err, tc.Expected.ErrorCode)
			}
		})
	}
}

// setupCreateEnvironmentMocks sets up mocks for CreateEnvironment tests
func (suite *EnvironmentServiceTestSuite) setupCreateEnvironmentMocks(mockSetup MockSetup) {
	if mockSetup.EnvironmentRepo.Method != "" {
		suite.setupEnvironmentRepoMock(mockSetup.EnvironmentRepo)
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

// setupListEnvironmentsMocks sets up mocks for ListEnvironments tests
func (suite *EnvironmentServiceTestSuite) setupListEnvironmentsMocks(mockSetup MockSetup) {
	if mockSetup.EnvironmentRepo.Method != "" {
		suite.setupEnvironmentRepoMock(mockSetup.EnvironmentRepo)
	}
}

// setupListMyEnvironmentsMocks sets up mocks for ListMyEnvironments tests
func (suite *EnvironmentServiceTestSuite) setupListMyEnvironmentsMocks(mockSetup MockSetup) {
	if mockSetup.IamService.Method != "" {
		suite.setupIamServiceMock(mockSetup.IamService)
	}
}

// setupGetEnvironmentMocks sets up mocks for GetEnvironment tests
func (suite *EnvironmentServiceTestSuite) setupGetEnvironmentMocks(mockSetup MockSetup) {
	if mockSetup.EnvironmentRepo.Method != "" {
		suite.setupEnvironmentRepoMock(mockSetup.EnvironmentRepo)
	}
}

// setupGetEnvironmentByNameMocks sets up mocks for GetEnvironmentByName tests
func (suite *EnvironmentServiceTestSuite) setupGetEnvironmentByNameMocks(mockSetup MockSetup) {
	if mockSetup.EnvironmentRepo.Method != "" {
		suite.setupEnvironmentRepoMock(mockSetup.EnvironmentRepo)
	}
}

// setupUpdateEnvironmentMocks sets up mocks for UpdateEnvironment tests
func (suite *EnvironmentServiceTestSuite) setupUpdateEnvironmentMocks(mockSetup MockSetup) {
	if mockSetup.EnvironmentRepo.Method != "" {
		suite.setupEnvironmentRepoMock(mockSetup.EnvironmentRepo)
	}
}

// setupDeleteEnvironmentMocks sets up mocks for DeleteEnvironment tests
func (suite *EnvironmentServiceTestSuite) setupDeleteEnvironmentMocks(mockSetup MockSetup) {
	if mockSetup.EnvironmentRepo.Method != "" {
		suite.setupEnvironmentRepoMock(mockSetup.EnvironmentRepo)
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
func (suite *EnvironmentServiceTestSuite) validateNegativeMocks(testCase string) {
	// CreateEnvironment error cases - these should NOT call downstream methods
	if strings.Contains(testCase, "invalid_user_id") {
		// When user ID is invalid, IAM service and policy enforcer should NOT be called
		suite.mockRepo.AssertNotCalled(suite.T(), "CreateEnvironment")
		suite.mockIamRepo.AssertNotCalled(suite.T(), "CreateRoleBinding")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "AddResourceOwner")
	}

	// CreateEnvironment database error - should NOT call IAM service or policy enforcer
	if strings.Contains(testCase, "database_error") {
		suite.mockIamRepo.AssertNotCalled(suite.T(), "CreateRoleBinding")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "AddResourceOwner")
	}

	// CreateEnvironment IAM error - should NOT call policy enforcer
	if strings.Contains(testCase, "iam_service_error") {
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "AddResourceOwner")
	}

	// DeleteEnvironment error cases
	if strings.Contains(testCase, "invalid_env_id") {
		suite.mockRepo.AssertNotCalled(suite.T(), "DeleteEnvironment")
		suite.mockIamRepo.AssertNotCalled(suite.T(), "DeleteRoleBinding")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "RemoveResource")
	}

	if strings.Contains(testCase, "environment_not_found") {
		suite.mockIamRepo.AssertNotCalled(suite.T(), "DeleteRoleBinding")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "RemoveResource")
	}
}

// setupEnvironmentRepoMock sets up the environment repository mock
func (suite *EnvironmentServiceTestSuite) setupEnvironmentRepoMock(config MockConfig) {
	switch config.Method {
	case "CreateEnvironment":
		if config.Return["error"] != nil {
			errorMsg := config.Return["error"].(string)
			var err error

			// Handle specific error types
			switch errorMsg {
			case "check constraint violation":
				// Create a PostgreSQL-style error for check constraint violation
				err = &postgresTestError{code: "23514", message: errorMsg}
			case "duplicate key value violates unique constraint":
				// Create a PostgreSQL-style error for unique constraint violation
				err = &postgresTestError{code: "23505", message: errorMsg}
			case "foreign key constraint violation":
				// Create a PostgreSQL-style error for foreign key constraint violation
				err = &postgresTestError{code: "23503", message: errorMsg}
			case "sql: no rows in result set":
				err = sql.ErrNoRows
			default:
				err = errors.New(errorMsg)
			}

			suite.mockRepo.On("CreateEnvironment", suite.ctx, mock.AnythingOfType("environmentdb.CreateEnvironmentParams")).
				Return(environmentdb.Environment{}, err).Once()
		} else {
			// Build mock environment from test data
			envData := config.Return["environment"].(map[string]interface{})
			env := environmentdb.Environment{
				ID:            uuid.MustParse(envData["id"].(string)),
				Name:          envData["name"].(string),
				SecretGroupID: uuid.MustParse(envData["secret_group_id"].(string)),
			}
			if envData["description"] != nil {
				desc := envData["description"].(string)
				env.Description = sql.NullString{String: desc, Valid: true}
			}
			env.CreatedAt = time.Now()
			env.UpdatedAt = time.Now()

			suite.mockRepo.On("CreateEnvironment", suite.ctx, mock.AnythingOfType("environmentdb.CreateEnvironmentParams")).
				Return(env, nil).Once()
		}
	case "ListEnvironmentsBySecretGroup":
		if config.Return["error"] != nil {
			suite.mockRepo.On("ListEnvironmentsBySecretGroup", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return([]environmentdb.Environment{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock environments from test data
			environments := []environmentdb.Environment{}
			if config.Return["environments"] != nil {
				for _, env := range config.Return["environments"].([]interface{}) {
					envMap := env.(map[string]interface{})
					envItem := environmentdb.Environment{
						ID:            uuid.MustParse(envMap["id"].(string)),
						Name:          envMap["name"].(string),
						SecretGroupID: uuid.MustParse(envMap["secret_group_id"].(string)),
					}
					if envMap["description"] != nil {
						desc := envMap["description"].(string)
						envItem.Description = sql.NullString{String: desc, Valid: true}
					}
					envItem.CreatedAt = time.Now()
					envItem.UpdatedAt = time.Now()
					environments = append(environments, envItem)
				}
			}
			suite.mockRepo.On("ListEnvironmentsBySecretGroup", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return(environments, nil).Once()
		}
	case "GetEnvironmentByID":
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

			suite.mockRepo.On("GetEnvironmentByID", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return(environmentdb.Environment{}, err).Once()
		} else {
			// Build mock environment from test data
			envData := config.Return["environment"].(map[string]interface{})
			env := environmentdb.Environment{
				ID:            uuid.MustParse(envData["id"].(string)),
				Name:          envData["name"].(string),
				SecretGroupID: uuid.MustParse(envData["secret_group_id"].(string)),
			}
			if envData["description"] != nil {
				desc := envData["description"].(string)
				env.Description = sql.NullString{String: desc, Valid: true}
			}
			env.CreatedAt = time.Now()
			env.UpdatedAt = time.Now()

			suite.mockRepo.On("GetEnvironmentByID", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return(env, nil).Once()
		}
	case "GetEnvironmentByName":
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

			suite.mockRepo.On("GetEnvironmentByName", suite.ctx, mock.AnythingOfType("environmentdb.GetEnvironmentByNameParams")).
				Return(environmentdb.GetEnvironmentByNameRow{}, err).Once()
		} else {
			// Build mock environment from test data
			envData := config.Return["environment"].(map[string]interface{})
			env := environmentdb.GetEnvironmentByNameRow{
				ID:            uuid.MustParse(envData["id"].(string)),
				Name:          envData["name"].(string),
				SecretGroupID: uuid.MustParse(envData["secret_group_id"].(string)),
			}
			if envData["description"] != nil {
				desc := envData["description"].(string)
				env.Description = sql.NullString{String: desc, Valid: true}
			}
			env.CreatedAt = time.Now()
			env.UpdatedAt = time.Now()

			suite.mockRepo.On("GetEnvironmentByName", suite.ctx, mock.AnythingOfType("environmentdb.GetEnvironmentByNameParams")).
				Return(env, nil).Once()
		}
	case "UpdateEnvironment":
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

			suite.mockRepo.On("UpdateEnvironment", suite.ctx, mock.AnythingOfType("environmentdb.UpdateEnvironmentParams")).
				Return(environmentdb.Environment{}, err).Once()
		} else {
			// Build mock environment from test data
			envData := config.Return["environment"].(map[string]interface{})
			env := environmentdb.Environment{
				ID:            uuid.MustParse(envData["id"].(string)),
				Name:          envData["name"].(string),
				SecretGroupID: uuid.MustParse(envData["secret_group_id"].(string)),
			}
			if envData["description"] != nil {
				desc := envData["description"].(string)
				env.Description = sql.NullString{String: desc, Valid: true}
			}
			env.CreatedAt = time.Now()
			env.UpdatedAt = time.Now()

			suite.mockRepo.On("UpdateEnvironment", suite.ctx, mock.AnythingOfType("environmentdb.UpdateEnvironmentParams")).
				Return(env, nil).Once()
		}
	case "DeleteEnvironment":
		if config.Return["error"] != nil {
			errorMsg := config.Return["error"].(string)
			var err error

			// Handle specific error types
			switch errorMsg {
			case "sql: no rows in result set":
				err = sql.ErrNoRows
			case "foreign key constraint violation":
				// Create a PostgreSQL-style error for foreign key constraint violation
				err = &postgresTestError{code: "23503", message: errorMsg}
			default:
				err = errors.New(errorMsg)
			}

			suite.mockRepo.On("DeleteEnvironment", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return(err).Once()
		} else {
			suite.mockRepo.On("DeleteEnvironment", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return(nil).Once()
		}
	}
}

// setupIamServiceMock sets up the IAM service mock
func (suite *EnvironmentServiceTestSuite) setupIamServiceMock(config MockConfig) {
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
	case "ListAccessibleEnvironments":
		if config.Return["error"] != nil {
			suite.mockIamRepo.On("ListAccessibleEnvironments", suite.ctx, mock.AnythingOfType("iam_db.ListAccessibleEnvironmentsParams")).
				Return([]iam_db.ListAccessibleEnvironmentsRow{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock accessible environments from test data
			environments := []iam_db.ListAccessibleEnvironmentsRow{}
			if config.Return["environments"] != nil {
				for _, env := range config.Return["environments"].([]interface{}) {
					envMap := env.(map[string]interface{})
					envItem := iam_db.ListAccessibleEnvironmentsRow{
						ID:              uuid.NullUUID{UUID: uuid.MustParse(envMap["id"].(string)), Valid: true},
						Name:            envMap["name"].(string),
						SecretGroupName: envMap["secret_group_name"].(string),
						Role:            iam_db.UserRole(envMap["role"].(string)),
					}
					environments = append(environments, envItem)
				}
			}
			suite.mockIamRepo.On("ListAccessibleEnvironments", suite.ctx, mock.AnythingOfType("iam_db.ListAccessibleEnvironmentsParams")).
				Return(environments, nil).Once()
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
func (suite *EnvironmentServiceTestSuite) setupPolicyEnforcerMock(config MockConfig) {
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

// buildCreateEnvironmentRequest builds a CreateEnvironmentRequest from test input
func (suite *EnvironmentServiceTestSuite) buildCreateEnvironmentRequest(input map[string]interface{}) CreateEnvironmentRequest {
	req := CreateEnvironmentRequest{
		Name:         input["name"].(string),
		SecretGroup:  input["secret_group"].(string),
		Organization: input["organization"].(string),
		UserId:       input["user_id"].(string),
	}
	if input["description"] != nil {
		req.Description = input["description"].(string)
	}
	return req
}

// TestEnvironmentServiceTestSuite runs the test suite
func TestEnvironmentServiceTestSuite(t *testing.T) {
	suite.Run(t, new(EnvironmentServiceTestSuite))
}
