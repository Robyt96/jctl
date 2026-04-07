# Tasks: Get Build Parameters

## 1. Update Data Models
- [x] 1.1 Add `BuildAction` struct to `internal/client/models.go` with `Parameters []Parameter` field
- [x] 1.2 Update `Build` struct to include `Actions []BuildAction` field
- [x] 1.3 Add `ExtractParameters()` method to `Build` struct that iterates through actions and collects all parameters

## 2. Enhance API Client
- [x] 2.1 Update `GetBuildInfo()` method in `internal/client/client.go` to include `actions[parameters[name,value]]` in the API tree parameter
- [x] 2.2 Verify the API request properly fetches parameter data from Jenkins
- [x] 2.3 Test that folder paths are correctly URL-encoded in the API request

## 3. Implement CLI Command
- [x] 3.1 Create `buildsParamsCmd` command in `cmd/jctl/builds.go` with proper usage, description, and examples
- [x] 3.2 Implement `runBuildsParams()` function to handle command execution
- [x] 3.3 Add argument validation to ensure exactly 2 arguments are provided
- [x] 3.4 Add build number validation to ensure it's a valid positive integer
- [x] 3.5 Implement API client call to retrieve build information
- [x] 3.6 Extract parameters from build using `ExtractParameters()` method
- [x] 3.7 Handle empty parameter list with appropriate message
- [x] 3.8 Register the command with the `buildsCmd` parent command

## 4. Implement Output Formatting
- [ ] 4.1 Create `formatParamsOutput()` function to route to appropriate formatter based on output format
- [ ] 4.2 Implement `formatParamsText()` for tabular text output with aligned columns
- [ ] 4.3 Implement `formatParamsJSON()` for JSON array output
- [ ] 4.4 Implement `formatParamsYAML()` for YAML document output
- [ ] 4.5 Ensure all formatters handle empty parameter lists gracefully

## 5. Implement Error Handling
- [ ] 5.1 Add error handling for pipeline not found (404) with descriptive message
- [ ] 5.2 Add error handling for build not found (404) with descriptive message
- [ ] 5.3 Add error handling for invalid build number format
- [ ] 5.4 Add error handling for missing/insufficient arguments with usage display
- [ ] 5.5 Add error handling for authentication failures
- [ ] 5.6 Ensure all error messages follow the established jctl pattern

## 6. Write Unit Tests
- [ ] 6.1 Test pipeline not found error case
- [ ] 6.2 Test build not found error case
- [ ] 6.3 Test invalid build number format error case
- [ ] 6.4 Test missing arguments error case
- [ ] 6.5 Test build with no parameters (empty actions array)
- [ ] 6.6 Test build with multiple parameter actions
- [ ] 6.7 Test parameters with special characters
- [ ] 6.8 Test text output formatting
- [ ] 6.9 Test JSON output formatting
- [ ] 6.10 Test YAML output formatting

## 7. Write Property-Based Tests
- [ ] 7.1 Set up gopter testing framework
- [ ] 7.2 [PBT] Property 1: Parameter retrieval for valid inputs (100+ iterations)
- [ ] 7.3 [PBT] Property 2: Folder path support with URL encoding (100+ iterations)
- [ ] 7.4 [PBT] Property 3: Complete parameter display (100+ iterations)
- [ ] 7.5 [PBT] Property 4: Text format structure (100+ iterations)
- [ ] 7.6 [PBT] Property 5: Serialization round-trip (100+ iterations)
- [ ] 7.7 [PBT] Property 6: Output format consistency (100+ iterations)
- [ ] 7.8 [PBT] Property 7: Input validation (100+ iterations)
- [ ] 7.9 [PBT] Property 8: Global flag support (100+ iterations)
- [ ] 7.10 [PBT] Property 9: Error response structure (100+ iterations)

## 8. Integration Testing
- [ ] 8.1 Test command with real Jenkins instance (manual testing)
- [ ] 8.2 Verify authentication works correctly
- [ ] 8.3 Test with various pipeline types (simple, folder-based, parameterized, non-parameterized)
- [ ] 8.4 Verify output formats match expected structure
- [ ] 8.5 Test global flags (--profile, --jenkins-url, --output, --verbose, --timeout)

## 9. Documentation
- [x] 9.1 Add command examples to README.md
- [x] 9.2 Document the new command in user documentation
- [x] 9.3 Add inline code comments for complex logic
- [x] 9.4 Update CHANGELOG.md with new feature

## 10. Code Review and Refinement
- [ ] 10.1 Run linter and fix any issues
- [ ] 10.2 Ensure code follows Go best practices
- [ ] 10.3 Verify error messages are user-friendly
- [ ] 10.4 Check for code duplication and refactor if needed
- [ ] 10.5 Ensure consistent naming conventions
