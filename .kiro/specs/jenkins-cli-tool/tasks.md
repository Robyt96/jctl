# Implementation Plan: jctl (Jenkins Control Tool)

## Overview

This implementation plan breaks down the jctl CLI tool into discrete, incremental tasks. Each task builds on previous work, with testing integrated throughout. The implementation uses Go for its excellent CLI tooling, single-binary distribution, and fast performance.

## Tasks

- [x] 1. Set up Go project structure and dependencies
  - Initialize Go module with `go mod init`
  - Add dependencies: cobra (CLI framework), viper (configuration), go-yaml (YAML parsing)
  - Create directory structure: `cmd/`, `internal/`, `pkg/`
  - Set up basic main.go entry point
  - _Requirements: 8.1, 8.2, 8.4_

- [x] 2. Implement configuration management
  - [x] 2.1 Create configuration data structures
    - Define Config struct with all configuration fields
    - Implement YAML struct tags for parsing
    - _Requirements: 5.1, 5.2_

  - [x] 2.2 Implement configuration loading and validation
    - Load config from file using viper
    - Implement validation for URLs, timeouts, and other fields
    - Handle missing config file gracefully
    - _Requirements: 5.1, 5.2, 5.3, 5.4_

  - [x] 2.3 Write property test for configuration validation
    - **Property 7: Configuration Validation**
    - **Validates: Requirements 5.4**

  - [x] 2.4 Implement configuration precedence (CLI flags override file)
    - Merge configuration from file, environment variables, and CLI flags
    - Ensure CLI flags take highest precedence
    - _Requirements: 5.5_

  - [x] 2.5 Write property test for configuration precedence
    - **Property 6: Configuration Precedence**
    - **Validates: Requirements 5.5**

- [x] 2a. Refactor configuration for profile support
  - [x] 2a.1 Update configuration data structures for profiles
    - Modify Config struct to support multiple named profiles
    - Add Profile struct with per-profile settings
    - Add default_profile field to Config
    - Update YAML parsing to handle profiles map
    - _Requirements: 9.1_

  - [x] 2a.2 Implement profile loading and selection
    - Load all profiles from config file
    - Implement GetProfile(name) to retrieve specific profile
    - Implement default profile fallback when no profile specified
    - Handle non-existent profile errors
    - _Requirements: 9.2, 9.3, 9.7_

  - [x] 2a.3 Write property test for profile configuration retrieval
    - **Property 14: Profile Configuration Retrieval**
    - **Validates: Requirements 9.1, 9.2**

  - [ ]* 2a.4 Write property test for default profile fallback
    - **Property 15: Default Profile Fallback**
    - **Validates: Requirements 9.3**

  - [x] 2a.5 Add --profile global flag to CLI
    - Add --profile flag to root command
    - Ensure profile flag is available to all subcommands
    - Update config loading to use specified profile
    - _Requirements: 9.2_

- [x] 3. Refactor authentication for profile support
  - [x] 3.1 Update token storage to support multiple profiles
    - Create Credentials struct with map of profile names to tokens
    - Change storage from ~/.jctl/token to ~/.jctl/credentials (YAML format)
    - Implement per-profile token storage and retrieval
    - Migrate existing token file if present
    - _Requirements: 9.5, 9.6_

  - [x] 3.2 Write property test for profile credential isolation
    - **Property 13: Profile Credential Isolation**
    - **Validates: Requirements 9.5, 9.6**

  - [x] 3.3 Update auth manager to accept profile parameter
    - Modify Login(profile) to store token for specific profile
    - Modify GetToken(profile) to retrieve token for specific profile
    - Update all auth methods to work with profile names
    - _Requirements: 9.5, 9.6_

  - [ ]* 3.4 Write property test for token persistence per profile
    - **Property 8: Token Persistence** (updated for profiles)
    - **Validates: Requirements 6.2, 6.5, 9.5**

  - [x] 3.5 Update auth login command to use current profile
    - Determine current profile from --profile flag or default
    - Store credentials under current profile name
    - Display which profile was authenticated
    - _Requirements: 9.5_

- [x] 4. Checkpoint - Ensure configuration and auth tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Implement Jenkins API client
  - [x] 5.1 Create HTTP client with authentication
    - Implement base HTTP client with timeout configuration
    - Add authentication header injection
    - Implement request/response logging for debugging
    - _Requirements: 6.5, 7.1_

  - [x] 5.2 Implement API endpoint methods
    - ListJobs() - GET /api/json?tree=jobs[name,url,color,description]
    - GetJob(name) - GET /job/{name}/api/json
    - ListBuilds(jobName) - GET /job/{name}/api/json?tree=builds[number,result,timestamp,duration,building]
    - GetBuildLog(jobName, buildNumber) - GET /job/{name}/{number}/consoleText
    - TriggerBuild(jobName, params) - POST /job/{name}/build or /job/{name}/buildWithParameters
    - GetBuildInfo(jobName, buildNumber) - GET /job/{name}/{number}/api/json
    - _Requirements: 1.1, 2.1, 3.1, 4.1_

  - [x] 5.3 Implement error handling and parsing
    - Parse Jenkins API error responses
    - Map HTTP status codes to user-friendly errors
    - Handle network errors with descriptive messages
    - _Requirements: 1.3, 7.1, 7.2, 7.3_

  - [x] 5.4 Implement progressive log streaming API methods
    - GetProgressiveLog(jobName, buildNumber, startByte) - GET /job/{name}/{number}/logText/progressiveText?start={startByte}
    - Parse X-Text-Size and X-More-Data headers
    - GetQueueItem(queueID) - GET /queue/item/{id}/api/json
    - Return ProgressiveLogResponse with content, nextOffset, hasMoreData
    - _Requirements: 3.5, 4.6_

  - [ ]* 5.5 Write property test for error message descriptiveness
    - **Property 10: Error Message Descriptiveness**
    - **Validates: Requirements 1.3, 2.3, 3.3, 4.5, 7.1, 7.2, 7.3, 7.4**

  - [ ]* 5.6 Write property test for non-existent resource handling
    - **Property 11: Non-Existent Resource Handling**
    - **Validates: Requirements 2.3, 3.3**

- [x] 6. Implement CLI commands structure
  - [x] 6.1 Set up Cobra command structure
    - Create root command with global flags (--jenkins-url, --config, --output, --verbose)
    - Implement --help and --version flags
    - Set up command groups: pipelines, builds, logs, trigger, auth, config
    - _Requirements: 8.1, 8.3, 8.4_

  - [ ]* 6.2 Write unit tests for help text
    - Test --help flag displays usage information
    - Test --version flag displays version
    - _Requirements: 8.1, 8.4_

  - [x] 6.3 Implement help text for insufficient arguments
    - Add validation to each command for required arguments
    - Display command-specific help when arguments are missing
    - _Requirements: 8.3_

  - [ ]* 6.4 Write property test for help text completeness
    - **Property 12: Help Text Completeness**
    - **Validates: Requirements 8.3**

- [x] 7. Implement pipelines list command
  - [x] 7.1 Create `jctl pipelines list` command handler
    - Call API client ListJobs()
    - Format output (text, JSON, YAML based on --output flag)
    - Handle empty pipeline list
    - _Requirements: 1.1, 1.2, 1.4_

  - [x] 7.2 Write property test for pipeline retrieval completeness
    - **Property 1: Pipeline Retrieval Completeness**
    - **Validates: Requirements 1.1, 1.2**

  - [ ]* 7.3 Write unit test for empty pipeline list
    - Test output when no pipelines exist
    - _Requirements: 1.4_

- [x] 8. Implement builds list command
  - [x] 8.1 Create `jctl builds list <pipeline>` command handler
    - Validate pipeline name argument is provided
    - Call API client ListBuilds()
    - Format output with build number, status, timestamp, duration
    - Handle empty build list and non-existent pipeline
    - _Requirements: 2.1, 2.2, 2.3, 2.4_

  - [ ]* 8.2 Write property test for build list completeness
    - **Property 2: Build List Completeness**
    - **Validates: Requirements 2.1, 2.2**

  - [ ]* 8.3 Write unit test for empty build list
    - Test output when pipeline has no builds
    - _Requirements: 2.4_

- [x] 9. Implement logs command
  - [x] 9.1 Create `jctl logs <pipeline> <build-number>` command handler
    - Validate pipeline and build number arguments
    - Call API client GetBuildLog()
    - Stream log output to stdout
    - Handle non-existent build
    - Handle in-progress builds (partial logs)
    - _Requirements: 3.1, 3.2, 3.3, 3.4_

  - [x] 9.2 Add --follow flag for progressive log streaming
    - Add --follow/-f flag to logs command
    - Implement progressive log polling using GetProgressiveLog()
    - Track byte offset between requests
    - Display new log content as it becomes available
    - Exit when build completes or user interrupts (Ctrl+C)
    - Handle signal interrupts gracefully
    - _Requirements: 3.5_

  - [ ]* 9.3 Write property test for progressive log completeness
    - **Property 17: Progressive Log Completeness**
    - **Validates: Requirements 3.5**

  - [ ]* 9.4 Write property test for log content completeness
    - **Property 3: Log Content Completeness**
    - **Validates: Requirements 3.1, 3.2**

  - [ ]* 9.5 Write unit test for in-progress build logs
    - Test partial log display for running builds
    - _Requirements: 3.4_

- [ ] 10. Checkpoint - Ensure read-only commands work end-to-end
  - Ensure all tests pass, ask the user if questions arise.

- [x] 11. Implement trigger command
  - [x] 11.1 Create `jctl trigger <pipeline>` command handler
    - Parse --param flags into parameter map
    - Validate pipeline exists
    - Call API client TriggerBuild()
    - Display confirmation with queue ID or build number
    - _Requirements: 4.1, 4.2_

  - [x] 11.2 Add --follow flag for progressive log streaming after trigger
    - Add --follow/-f flag to trigger command
    - After triggering, poll queue item until build starts
    - Once build starts, retrieve build number
    - Stream build logs progressively using GetProgressiveLog()
    - Display final build status when complete
    - Handle signal interrupts gracefully
    - _Requirements: 4.6_

  - [ ]* 11.3 Write property test for build trigger success
    - **Property 4: Build Trigger Success**
    - **Validates: Requirements 4.1, 4.2**

  - [x] 11.4 Implement parameter validation
    - Fetch pipeline definition to get required parameters
    - Validate all required parameters are provided
    - List missing parameters in error message
    - Handle default parameter values
    - _Requirements: 4.3, 4.4_

  - [ ]* 11.5 Write property test for parameter validation
    - **Property 5: Parameter Validation**
    - **Validates: Requirements 4.3**

  - [ ]* 11.6 Write unit test for default parameter handling
    - Test triggering parameterized pipeline without params uses defaults
    - _Requirements: 4.4_

- [x] 12. Implement auth login command
  - [x] 12.1 Create `jctl auth login` command handler
    - Initiate OAuth flow or prompt for API token
    - Store received token using auth manager
    - Display success message
    - _Requirements: 6.1, 6.2, 6.4_

  - [ ]* 12.2 Write unit tests for auth command
    - Test token storage after successful login
    - Test error handling for failed authentication
    - _Requirements: 6.2, 6.4_

- [ ] 13. Implement config show command
  - [ ] 13.1 Create `jctl config show` command handler
    - Display current effective configuration
    - Show source of each config value (file, env, flag, default)
    - Format output clearly
    - _Requirements: 5.1, 5.5_

  - [ ]* 13.2 Write unit test for config show
    - Test output includes all config values
    - _Requirements: 5.1_

- [x] 13a. Implement profile management commands
  - [x] 13a.1 Create `jctl profile list` command handler
    - Retrieve all configured profiles from config manager
    - Display profile names with Jenkins URLs
    - Indicate which profile is the default
    - Format output based on --output flag
    - _Requirements: 9.4_

  - [ ]* 13a.2 Write property test for profile listing completeness
    - **Property 16: Profile Listing Completeness**
    - **Validates: Requirements 9.4**

  - [x] 13a.3 Create `jctl profile show [profile-name]` command handler
    - Show specified profile's configuration (or current if not specified)
    - Display all profile settings (URL, timeout, auth method, etc.)
    - Show whether credentials are configured for the profile
    - Handle non-existent profile error
    - _Requirements: 9.1, 9.7_

  - [x] 13a.4 Create `jctl profile set-default <profile-name>` command handler
    - Validate profile exists
    - Update default_profile in config file
    - Save updated config file
    - Display confirmation message
    - _Requirements: 9.3_

  - [ ]* 13a.5 Write unit tests for profile commands
    - Test profile list output
    - Test profile show output
    - Test set-default updates config
    - Test error handling for non-existent profiles
    - _Requirements: 9.4, 9.7_

- [ ] 14. Implement output formatting
  - [ ] 14.1 Create output formatter interface
    - Implement text formatter (human-readable tables)
    - Implement JSON formatter
    - Implement YAML formatter
    - Apply formatter based on --output flag
    - _Requirements: 1.2, 2.2, 3.2, 4.2, 7.5_

  - [ ]* 14.2 Write unit tests for each output format
    - Test text, JSON, and YAML formatting
    - _Requirements: 1.2, 2.2_

- [ ] 15. Add error handling and user feedback
  - [ ] 15.1 Implement consistent error formatting
    - Create error message formatter with Error/Details/Suggestion format
    - Apply to all error paths
    - _Requirements: 7.1, 7.2, 7.3, 7.4_

  - [ ] 15.2 Implement success confirmation messages
    - Add confirmation output for all successful operations
    - _Requirements: 7.5_

  - [ ] 15.3 Implement exit codes
    - Set appropriate exit codes for different error types
    - _Requirements: 7.1, 7.2, 7.3, 7.4_

- [ ] 16. Final integration and polish
  - [ ] 16.1 Add retry logic for transient errors
    - Implement exponential backoff for network timeouts
    - Handle rate limiting with Retry-After header
    - _Requirements: 7.1_

  - [ ] 16.2 Add verbose logging mode
    - Implement --verbose flag to show HTTP requests/responses
    - Log configuration loading details
    - _Requirements: 8.1_

  - [ ]* 16.3 Write integration tests
    - Test end-to-end flows with mock Jenkins server
    - Test configuration loading and usage
    - Test authentication flow
    - _Requirements: All_

- [ ] 17. Final checkpoint - Ensure all tests pass
  - Run full test suite
  - Verify all property tests pass with 100+ iterations
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Property tests validate universal correctness properties across random inputs
- Unit tests validate specific examples and edge cases
- Integration tests verify end-to-end functionality
- The implementation follows Go best practices and idioms
- Use Go modules for dependency management
- Follow standard Go project layout (cmd/, internal/, pkg/)
