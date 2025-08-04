package org

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
	orgdb "github.com/Gkemhcs/kavach-backend/internal/org/gen"
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
	Success bool        `json:"success"`
	Error   interface{} `json:"error"`
	// Additional fields for specific test types
	Organization  interface{}              `json:"organization,omitempty"`
	Organizations []map[string]interface{} `json:"organizations,omitempty"`
}

// MockSetup represents the mock configuration for a test
type MockSetup struct {
	OrgRepo        MockConfig `json:"org_repo,omitempty"`
	IamService     MockConfig `json:"iam_service,omitempty"`
	PolicyEnforcer MockConfig `json:"policy_enforcer,omitempty"`
}

// MockConfig represents a single mock configuration
type MockConfig struct {
	Method string                 `json:"method"`
	Return map[string]interface{} `json:"return"`
}

// OrgServiceTestSuite provides a test suite for the Organization service
type OrgServiceTestSuite struct {
	suite.Suite
	service            *OrganizationService
	mockRepo           *MockOrgRepository
	mockIamRepo        *iam.MockIamRepository
	mockUserResolver   *MockUserResolver
	mockGroupResolver  *MockUserGroupResolver
	mockPolicyEnforcer *MockPolicyEnforcer
	iamService         *iam.IamService
	logger             *logrus.Logger
	ctx                context.Context
}

// MockUserResolver is a mock implementation of types.UserResolver
type MockUserResolver struct {
	mock.Mock
}

// GetUserInfoByGithubUserName mocks the GetUserInfoByGithubUserName method
func (m *MockUserResolver) GetUserInfoByGithubUserName(ctx context.Context, username string) (*userdb.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userdb.User), args.Error(1)
}

// MockUserGroupResolver is a mock implementation of types.UserGroupResolver
type MockUserGroupResolver struct {
	mock.Mock
}

// GetUserGroupByName mocks the GetUserGroupByName method
func (m *MockUserGroupResolver) GetUserGroupByName(ctx context.Context, groupName, orgID string) (*groupsdb.UserGroup, error) {
	args := m.Called(ctx, groupName, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*groupsdb.UserGroup), args.Error(1)
}

// MockPolicyEnforcer is a mock implementation of PolicyEnforcer using authz.MockEnforcer
type MockPolicyEnforcer struct {
	*authz.MockEnforcer
}

// SetupSuite initializes the test suite
func (suite *OrgServiceTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.logger = logrus.New()
	suite.logger.SetLevel(logrus.DebugLevel)
	suite.logger.SetOutput(io.Discard) // Discard logs during tests to keep output clean
}

// SetupTest initializes each test
func (suite *OrgServiceTestSuite) SetupTest() {
	// Create new mock instances for each test to ensure isolation
	suite.mockRepo = &MockOrgRepository{}
	suite.mockIamRepo = &iam.MockIamRepository{}
	suite.mockUserResolver = &MockUserResolver{}
	suite.mockGroupResolver = &MockUserGroupResolver{}
	suite.mockPolicyEnforcer = &MockPolicyEnforcer{
		MockEnforcer: &authz.MockEnforcer{},
	}

	// Create concrete IAM service with mock repository
	suite.iamService = iam.NewIamService(
		suite.mockIamRepo,
		suite.mockUserResolver,
		suite.mockGroupResolver,
		suite.logger,
		suite.mockPolicyEnforcer,
	)

	suite.service = NewOrganizationService(
		suite.mockRepo,
		suite.logger,
		*suite.iamService,
		suite.mockPolicyEnforcer,
	)
}

// TearDownTest cleans up after each test
func (suite *OrgServiceTestSuite) TearDownTest() {
	// Assert all mock expectations were met
	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockIamRepo.AssertExpectations(suite.T())
	suite.mockUserResolver.AssertExpectations(suite.T())
	suite.mockGroupResolver.AssertExpectations(suite.T())
	suite.mockPolicyEnforcer.MockEnforcer.AssertExpectations(suite.T())

	// Reset all mocks to ensure clean state for next test
	suite.mockRepo.ExpectedCalls = nil
	suite.mockIamRepo.ExpectedCalls = nil
	suite.mockUserResolver.ExpectedCalls = nil
	suite.mockGroupResolver.ExpectedCalls = nil
	suite.mockPolicyEnforcer.MockEnforcer.ExpectedCalls = nil
}

// loadTestData loads test cases from embedded JSON files
func (suite *OrgServiceTestSuite) loadTestData(filename string) *TestData {
	data, err := testDataFS.ReadFile("test_data/" + filename)
	require.NoError(suite.T(), err, "Failed to read embedded test data file: %s", filename)

	var testData TestData
	err = json.Unmarshal(data, &testData)
	require.NoError(suite.T(), err, "Failed to unmarshal test data from: %s", filename)

	return &testData
}

// TestCreateOrganizationWithData runs CreateOrganization tests using data-driven approach
func (suite *OrgServiceTestSuite) TestCreateOrganizationWithData() {
	testData := suite.loadTestData("create_organization_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			suite.setupCreateOrganizationMocks(tc.MockSetup)

			req := suite.buildCreateOrganizationRequest(tc.Input)
			result, err := suite.service.CreateOrganization(suite.ctx, req)

			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected organization result but got nil")
				if tc.Expected.Organization != nil {
					expectedOrg := tc.Expected.Organization.(map[string]interface{})
					assert.Equal(suite.T(), expectedOrg["name"].(string), result.Name, "Organization name mismatch")
					if expectedOrg["description"] != nil && result.Description.Valid {
						assert.Equal(suite.T(), expectedOrg["description"].(string), result.Description.String, "Organization description mismatch")
					}
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

// TestListOrganizationsWithData runs ListOrganizations tests using data-driven approach
func (suite *OrgServiceTestSuite) TestListOrganizationsWithData() {
	testData := suite.loadTestData("list_organizations_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			suite.setupListOrganizationsMocks(tc.MockSetup)

			userID := tc.Input["user_id"].(string)
			result, err := suite.service.ListOrganizations(suite.ctx, userID)

			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				if tc.Expected.Organizations != nil {
					assert.Len(suite.T(), result, len(tc.Expected.Organizations), "Expected %d organizations, got %d", len(tc.Expected.Organizations), len(result))
					// Validate each organization's fields
					for i, expectedOrg := range tc.Expected.Organizations {
						if i < len(result) {
							assert.Equal(suite.T(), expectedOrg["name"].(string), result[i].Name, "Organization name mismatch at index %d", i)
							if expectedOrg["description"] != nil && result[i].Description.Valid {
								assert.Equal(suite.T(), expectedOrg["description"].(string), result[i].Description.String, "Organization description mismatch at index %d", i)
							}
						}
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

// TestListMyOrganizationsWithData runs ListMyOrganizations tests using data-driven approach
func (suite *OrgServiceTestSuite) TestListMyOrganizationsWithData() {
	testData := suite.loadTestData("list_my_organizations_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			suite.setupListMyOrganizationsMocks(tc.MockSetup)

			userID := tc.Input["user_id"].(string)
			result, err := suite.service.ListMyOrganizations(suite.ctx, userID)

			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				if tc.Expected.Organizations != nil {
					assert.Len(suite.T(), result, len(tc.Expected.Organizations), "Expected %d organizations, got %d", len(tc.Expected.Organizations), len(result))
					// Validate each organization's fields
					for i, expectedOrg := range tc.Expected.Organizations {
						if i < len(result) {
							assert.Equal(suite.T(), expectedOrg["organization_id"].(string), result[i].ID.String(), "Organization ID mismatch at index %d", i)
							assert.Equal(suite.T(), expectedOrg["organization_name"].(string), result[i].OrgName, "Organization name mismatch at index %d", i)
							assert.Equal(suite.T(), expectedOrg["role"].(string), string(result[i].Role), "Organization role mismatch at index %d", i)
						}
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

// TestGetOrganizationWithData runs GetOrganization tests using data-driven approach
func (suite *OrgServiceTestSuite) TestGetOrganizationWithData() {
	testData := suite.loadTestData("get_organization_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			suite.setupGetOrganizationMocks(tc.MockSetup)

			userID := tc.Input["user_id"].(string)
			orgID := tc.Input["org_id"].(string)
			result, err := suite.service.GetOrganization(suite.ctx, userID, orgID)

			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected organization result but got nil")
				if tc.Expected.Organization != nil {
					expectedOrg := tc.Expected.Organization.(map[string]interface{})
					assert.Equal(suite.T(), expectedOrg["name"].(string), result.Name, "Organization name mismatch")
					if expectedOrg["description"] != nil && result.Description.Valid {
						assert.Equal(suite.T(), expectedOrg["description"].(string), result.Description.String, "Organization description mismatch")
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

// TestGetOrganizationByNameWithData runs GetOrganizationByName tests using data-driven approach
func (suite *OrgServiceTestSuite) TestGetOrganizationByNameWithData() {
	testData := suite.loadTestData("get_organization_by_name_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			suite.setupGetOrganizationByNameMocks(tc.MockSetup)

			orgName := tc.Input["org_name"].(string)
			result, err := suite.service.GetOrganizationByName(suite.ctx, orgName)

			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected organization result but got nil")
				if tc.Expected.Organization != nil {
					expectedOrg := tc.Expected.Organization.(map[string]interface{})
					assert.Equal(suite.T(), expectedOrg["name"].(string), result.Name, "Organization name mismatch")
					if expectedOrg["description"] != nil && result.Description.Valid {
						assert.Equal(suite.T(), expectedOrg["description"].(string), result.Description.String, "Organization description mismatch")
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

// TestUpdateOrganizationWithData runs UpdateOrganization tests using data-driven approach
func (suite *OrgServiceTestSuite) TestUpdateOrganizationWithData() {
	testData := suite.loadTestData("update_organization_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			suite.setupUpdateOrganizationMocks(tc.MockSetup)

			userID := tc.Input["user_id"].(string)
			orgID := tc.Input["org_id"].(string)
			req := suite.buildUpdateOrganizationRequest(tc.Input)
			result, err := suite.service.UpdateOrganization(suite.ctx, userID, orgID, req)

			if tc.Expected.Success {
				require.NoError(suite.T(), err, "Expected success but got error: %v", err)
				require.NotNil(suite.T(), result, "Expected organization result but got nil")
				if tc.Expected.Organization != nil {
					expectedOrg := tc.Expected.Organization.(map[string]interface{})
					assert.Equal(suite.T(), expectedOrg["name"].(string), result.Name, "Organization name mismatch")
					if expectedOrg["description"] != nil && result.Description.Valid {
						assert.Equal(suite.T(), expectedOrg["description"].(string), result.Description.String, "Organization description mismatch")
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

// TestDeleteOrganizationWithData runs DeleteOrganization tests using data-driven approach
func (suite *OrgServiceTestSuite) TestDeleteOrganizationWithData() {
	testData := suite.loadTestData("delete_organization_test_cases.json")

	for _, tc := range testData.TestCases {
		suite.Run(tc.Name, func() {
			suite.setupDeleteOrganizationMocks(tc.MockSetup)

			orgID := tc.Input["org_id"].(string)
			err := suite.service.DeleteOrganization(suite.ctx, orgID)

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

// setupCreateOrganizationMocks sets up mocks for CreateOrganization tests
func (suite *OrgServiceTestSuite) setupCreateOrganizationMocks(mockSetup MockSetup) {
	if mockSetup.OrgRepo.Method != "" {
		suite.setupOrgRepoMock(mockSetup.OrgRepo)
	}

	if mockSetup.IamService.Method != "" {
		suite.setupIamServiceMock(mockSetup.IamService)
	}

	if mockSetup.PolicyEnforcer.Method != "" {
		suite.setupPolicyEnforcerMock(mockSetup.PolicyEnforcer)
	}
}

// setupListOrganizationsMocks sets up mocks for ListOrganizations tests
func (suite *OrgServiceTestSuite) setupListOrganizationsMocks(mockSetup MockSetup) {
	if mockSetup.OrgRepo.Method != "" {
		suite.setupOrgRepoMock(mockSetup.OrgRepo)
	}
}

// setupListMyOrganizationsMocks sets up mocks for ListMyOrganizations tests
func (suite *OrgServiceTestSuite) setupListMyOrganizationsMocks(mockSetup MockSetup) {
	if mockSetup.IamService.Method != "" {
		suite.setupIamServiceMock(mockSetup.IamService)
	}
}

// setupGetOrganizationMocks sets up mocks for GetOrganization tests
func (suite *OrgServiceTestSuite) setupGetOrganizationMocks(mockSetup MockSetup) {
	if mockSetup.OrgRepo.Method != "" {
		suite.setupOrgRepoMock(mockSetup.OrgRepo)
	}
}

// setupGetOrganizationByNameMocks sets up mocks for GetOrganizationByName tests
func (suite *OrgServiceTestSuite) setupGetOrganizationByNameMocks(mockSetup MockSetup) {
	if mockSetup.OrgRepo.Method != "" {
		suite.setupOrgRepoMock(mockSetup.OrgRepo)
	}
}

// setupUpdateOrganizationMocks sets up mocks for UpdateOrganization tests
func (suite *OrgServiceTestSuite) setupUpdateOrganizationMocks(mockSetup MockSetup) {
	if mockSetup.OrgRepo.Method != "" {
		suite.setupOrgRepoMock(mockSetup.OrgRepo)
	}
}

// setupDeleteOrganizationMocks sets up mocks for DeleteOrganization tests
func (suite *OrgServiceTestSuite) setupDeleteOrganizationMocks(mockSetup MockSetup) {
	if mockSetup.OrgRepo.Method != "" {
		suite.setupOrgRepoMock(mockSetup.OrgRepo)
	}

	if mockSetup.IamService.Method != "" {
		suite.setupIamServiceMock(mockSetup.IamService)
	}

	if mockSetup.PolicyEnforcer.Method != "" {
		suite.setupPolicyEnforcerMock(mockSetup.PolicyEnforcer)
	}
}

// validateNegativeMocks validates that certain methods were NOT called when they shouldn't be
func (suite *OrgServiceTestSuite) validateNegativeMocks(testCase string) {
	// CreateOrganization error cases - these should NOT call downstream methods
	if testCase == "invalid_user_id" {
		// When user ID is invalid, IAM service and policy enforcer should NOT be called
		suite.mockRepo.AssertNotCalled(suite.T(), "CreateOrganization")
		suite.mockIamRepo.AssertNotCalled(suite.T(), "CreateRoleBinding")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "GrantRoleWithPermissions")
	}

	// CreateOrganization database error - should NOT call IAM service or policy enforcer
	if testCase == "database_error" {
		suite.mockIamRepo.AssertNotCalled(suite.T(), "CreateRoleBinding")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "GrantRoleWithPermissions")
	}

	// CreateOrganization IAM error - should NOT call policy enforcer
	if testCase == "iam_service_error" {
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "GrantRoleWithPermissions")
	}

	// DeleteOrganization error cases
	if testCase == "invalid_org_id" {
		// When org ID is invalid, IAM service and policy enforcer should NOT be called
		suite.mockRepo.AssertNotCalled(suite.T(), "DeleteOrganization")
		suite.mockIamRepo.AssertNotCalled(suite.T(), "DeleteRoleBinding")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "RemoveResource")
	}

	// DeleteOrganization database error - should NOT call IAM service or policy enforcer
	if testCase == "database_error" {
		suite.mockIamRepo.AssertNotCalled(suite.T(), "DeleteRoleBinding")
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "RemoveResource")
	}

	// DeleteOrganization IAM error - should NOT call policy enforcer
	if testCase == "iam_service_error" {
		suite.mockPolicyEnforcer.MockEnforcer.AssertNotCalled(suite.T(), "RemoveResource")
	}
}

// setupOrgRepoMock sets up organization repository mock
func (suite *OrgServiceTestSuite) setupOrgRepoMock(config MockConfig) {
	switch config.Method {
	case "CreateOrganization":
		if config.Return["error"] != nil {
			suite.mockRepo.On("CreateOrganization", suite.ctx, mock.AnythingOfType("orgdb.CreateOrganizationParams")).
				Return(orgdb.Organization{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock organization from test data
			orgData := config.Return["organization"].(map[string]interface{})
			org := orgdb.Organization{
				ID:      uuid.MustParse(orgData["id"].(string)),
				Name:    orgData["name"].(string),
				OwnerID: uuid.MustParse(orgData["owner_id"].(string)),
			}
			if orgData["description"] != nil {
				desc := orgData["description"].(string)
				org.Description = sql.NullString{String: desc, Valid: true}
			}
			org.CreatedAt = time.Now()
			org.UpdatedAt = time.Now()

			suite.mockRepo.On("CreateOrganization", suite.ctx, mock.AnythingOfType("orgdb.CreateOrganizationParams")).
				Return(org, nil).Once()
		}
	case "ListOrganizationsByOwner":
		if config.Return["error"] != nil {
			suite.mockRepo.On("ListOrganizationsByOwner", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return([]orgdb.Organization{}, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock organizations from test data
			orgs := []orgdb.Organization{}
			if config.Return["organizations"] != nil {
				for _, org := range config.Return["organizations"].([]interface{}) {
					orgMap := org.(map[string]interface{})
					orgItem := orgdb.Organization{
						ID:      uuid.MustParse(orgMap["id"].(string)),
						Name:    orgMap["name"].(string),
						OwnerID: uuid.MustParse(orgMap["owner_id"].(string)),
					}
					if orgMap["description"] != nil {
						desc := orgMap["description"].(string)
						orgItem.Description = sql.NullString{String: desc, Valid: true}
					}
					orgItem.CreatedAt = time.Now()
					orgItem.UpdatedAt = time.Now()
					orgs = append(orgs, orgItem)
				}
			}
			suite.mockRepo.On("ListOrganizationsByOwner", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return(orgs, nil).Once()
		}
	case "GetOrganizationByID":
		if config.Return["error"] != nil {
			errorMsg := config.Return["error"].(string)
			if errorMsg == "sql: no rows in result set" {
				suite.mockRepo.On("GetOrganizationByID", suite.ctx, mock.AnythingOfType("uuid.UUID")).
					Return(orgdb.Organization{}, sql.ErrNoRows).Once()
			} else {
				suite.mockRepo.On("GetOrganizationByID", suite.ctx, mock.AnythingOfType("uuid.UUID")).
					Return(orgdb.Organization{}, errors.New(errorMsg)).Once()
			}
		} else {
			// Build mock organization from test data
			orgData := config.Return["organization"].(map[string]interface{})
			org := orgdb.Organization{
				ID:      uuid.MustParse(orgData["id"].(string)),
				Name:    orgData["name"].(string),
				OwnerID: uuid.MustParse(orgData["owner_id"].(string)),
			}
			if orgData["description"] != nil {
				desc := orgData["description"].(string)
				org.Description = sql.NullString{String: desc, Valid: true}
			}
			org.CreatedAt = time.Now()
			org.UpdatedAt = time.Now()

			suite.mockRepo.On("GetOrganizationByID", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return(org, nil).Once()
		}
	case "GetOrganizationByName":
		if config.Return["error"] != nil {
			errorMsg := config.Return["error"].(string)
			if errorMsg == "sql: no rows in result set" {
				suite.mockRepo.On("GetOrganizationByName", suite.ctx, mock.AnythingOfType("string")).
					Return(orgdb.Organization{}, sql.ErrNoRows).Once()
			} else {
				suite.mockRepo.On("GetOrganizationByName", suite.ctx, mock.AnythingOfType("string")).
					Return(orgdb.Organization{}, errors.New(errorMsg)).Once()
			}
		} else {
			// Build mock organization from test data
			orgData := config.Return["organization"].(map[string]interface{})
			org := orgdb.Organization{
				ID:      uuid.MustParse(orgData["id"].(string)),
				Name:    orgData["name"].(string),
				OwnerID: uuid.MustParse(orgData["owner_id"].(string)),
			}
			if orgData["description"] != nil {
				desc := orgData["description"].(string)
				org.Description = sql.NullString{String: desc, Valid: true}
			}
			org.CreatedAt = time.Now()
			org.UpdatedAt = time.Now()

			suite.mockRepo.On("GetOrganizationByName", suite.ctx, mock.AnythingOfType("string")).
				Return(org, nil).Once()
		}
	case "UpdateOrganization":
		if config.Return["error"] != nil {
			errorMsg := config.Return["error"].(string)
			if errorMsg == "sql: no rows in result set" {
				suite.mockRepo.On("UpdateOrganization", suite.ctx, mock.AnythingOfType("orgdb.UpdateOrganizationParams")).
					Return(orgdb.Organization{}, sql.ErrNoRows).Once()
			} else {
				suite.mockRepo.On("UpdateOrganization", suite.ctx, mock.AnythingOfType("orgdb.UpdateOrganizationParams")).
					Return(orgdb.Organization{}, errors.New(errorMsg)).Once()
			}
		} else {
			// Build mock organization from test data
			orgData := config.Return["organization"].(map[string]interface{})
			org := orgdb.Organization{
				ID:      uuid.MustParse(orgData["id"].(string)),
				Name:    orgData["name"].(string),
				OwnerID: uuid.MustParse(orgData["owner_id"].(string)),
			}
			if orgData["description"] != nil {
				desc := orgData["description"].(string)
				org.Description = sql.NullString{String: desc, Valid: true}
			}
			org.CreatedAt = time.Now()
			org.UpdatedAt = time.Now()

			suite.mockRepo.On("UpdateOrganization", suite.ctx, mock.AnythingOfType("orgdb.UpdateOrganizationParams")).
				Return(org, nil).Once()
		}
	case "DeleteOrganization":
		if config.Return["error"] != nil {
			errorMsg := config.Return["error"].(string)
			if errorMsg == "sql: no rows in result set" {
				suite.mockRepo.On("DeleteOrganization", suite.ctx, mock.AnythingOfType("uuid.UUID")).
					Return(sql.ErrNoRows).Once()
			} else {
				suite.mockRepo.On("DeleteOrganization", suite.ctx, mock.AnythingOfType("uuid.UUID")).
					Return(errors.New(errorMsg)).Once()
			}
		} else {
			suite.mockRepo.On("DeleteOrganization", suite.ctx, mock.AnythingOfType("uuid.UUID")).
				Return(nil).Once()
		}
	}
}

// setupIamServiceMock sets up IAM repository mock for IAM service calls
func (suite *OrgServiceTestSuite) setupIamServiceMock(config MockConfig) {
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

			suite.mockIamRepo.On("CreateRoleBinding", suite.ctx, mock.AnythingOfType("iam_db.CreateRoleBindingParams")).
				Return(binding, nil).Once()
		}
	case "ListAccessibleOrganizations":
		if config.Return["error"] != nil {
			suite.mockIamRepo.On("ListAccessibleOrganizations", suite.ctx, mock.AnythingOfType("uuid.NullUUID")).
				Return(nil, errors.New(config.Return["error"].(string))).Once()
		} else {
			// Build mock organizations from test data
			orgs := []iam_db.ListAccessibleOrganizationsRow{}
			if config.Return["organizations"] != nil {
				for _, org := range config.Return["organizations"].([]interface{}) {
					orgMap := org.(map[string]interface{})
					orgs = append(orgs, iam_db.ListAccessibleOrganizationsRow{
						ID:      uuid.MustParse(orgMap["organization_id"].(string)),
						OrgName: orgMap["organization_name"].(string),
						Role:    iam_db.UserRole(orgMap["role"].(string)),
					})
				}
			}
			suite.mockIamRepo.On("ListAccessibleOrganizations", suite.ctx, mock.AnythingOfType("uuid.NullUUID")).
				Return(orgs, nil).Once()
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

// setupPolicyEnforcerMock sets up policy enforcer mock
func (suite *OrgServiceTestSuite) setupPolicyEnforcerMock(config MockConfig) {
	switch config.Method {
	case "GrantRoleWithPermissions":
		if config.Return["error"] != nil {
			suite.mockPolicyEnforcer.MockEnforcer.On("GrantRoleWithPermissions", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(errors.New(config.Return["error"].(string))).Once()
		} else {
			suite.mockPolicyEnforcer.MockEnforcer.On("GrantRoleWithPermissions", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
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
	}
}

// buildCreateOrganizationRequest builds a CreateOrganizationRequest from test input
func (suite *OrgServiceTestSuite) buildCreateOrganizationRequest(input map[string]interface{}) CreateOrganizationRequest {
	req := CreateOrganizationRequest{
		Name:   input["name"].(string),
		UserID: input["user_id"].(string),
	}

	if input["description"] != nil {
		req.Description = input["description"].(string)
	}

	return req
}

// buildUpdateOrganizationRequest builds an UpdateOrganizationRequest from test input
func (suite *OrgServiceTestSuite) buildUpdateOrganizationRequest(input map[string]interface{}) UpdateOrganizationRequest {
	req := UpdateOrganizationRequest{
		Name:   input["name"].(string),
		UserID: input["user_id"].(string),
	}

	if input["description"] != nil {
		req.Description = input["description"].(string)
	}

	return req
}

// TestOrgServiceTestSuite runs the test suite
func TestOrgServiceTestSuite(t *testing.T) {
	suite.Run(t, new(OrgServiceTestSuite))
}
