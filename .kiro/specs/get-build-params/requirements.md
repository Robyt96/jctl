# Requirements Document

## Introduction

This feature adds a new command to the jctl tool that retrieves and displays the parameters that were passed to a specific build. This allows users to inspect what parameter values were used when a build was triggered, which is useful for debugging, auditing, and reproducing builds.

## Glossary

- **jctl**: The Jenkins Control Tool CLI application
- **Build_Params_Command**: The new CLI command that retrieves build parameters
- **Jenkins_API_Client**: The internal client package that communicates with Jenkins REST API
- **Build**: A specific execution instance of a Jenkins pipeline, identified by pipeline name and build number
- **Build_Parameter**: A key-value pair passed to a build when it was triggered
- **Pipeline**: A Jenkins job that can be executed with or without parameters

## Requirements

### Requirement 1: Retrieve Build Parameters

**User Story:** As a developer, I want to retrieve the parameters used for a specific build, so that I can understand what configuration was used for that execution.

#### Acceptance Criteria

1. WHEN a valid pipeline name and build number are provided, THE Build_Params_Command SHALL retrieve the parameters from the Jenkins API
2. WHEN the specified build does not exist, THE Build_Params_Command SHALL return an error message indicating the build was not found
3. WHEN the specified pipeline does not exist, THE Build_Params_Command SHALL return an error message indicating the pipeline was not found
4. WHEN the build has no parameters, THE Build_Params_Command SHALL display a message indicating no parameters were used
5. THE Build_Params_Command SHALL support pipeline names with folder paths in the format "folder/subfolder/pipeline"

### Requirement 2: Display Build Parameters

**User Story:** As a developer, I want to see build parameters in a readable format, so that I can quickly understand the build configuration.

#### Acceptance Criteria

1. WHEN parameters are retrieved successfully, THE Build_Params_Command SHALL display each parameter name and value
2. WHERE the output format is "text", THE Build_Params_Command SHALL display parameters in a tabular format with aligned columns
3. WHERE the output format is "json", THE Build_Params_Command SHALL output parameters as a JSON array
4. WHERE the output format is "yaml", THE Build_Params_Command SHALL output parameters as a YAML document
5. THE Build_Params_Command SHALL respect the global output format flag consistent with other jctl commands

### Requirement 3: Command Line Interface

**User Story:** As a developer, I want a simple command syntax, so that I can quickly retrieve build parameters without complex options.

#### Acceptance Criteria

1. THE Build_Params_Command SHALL accept exactly two arguments: pipeline name and build number
2. WHEN insufficient arguments are provided, THE Build_Params_Command SHALL display usage information with examples
3. WHEN the build number is not a valid integer, THE Build_Params_Command SHALL return an error message
4. THE Build_Params_Command SHALL support all global flags including profile, jenkins-url, output, verbose, and timeout
5. THE Build_Params_Command SHALL follow the same authentication mechanism as other jctl commands

### Requirement 4: API Integration

**User Story:** As a developer, I want the command to use the existing Jenkins API client, so that authentication and error handling are consistent with other commands.

#### Acceptance Criteria

1. THE Jenkins_API_Client SHALL provide a method to retrieve build information including parameters
2. WHEN retrieving build information, THE Jenkins_API_Client SHALL include parameter data in the API request
3. THE Jenkins_API_Client SHALL handle folder paths by URL-encoding each path segment
4. WHEN the API request fails, THE Jenkins_API_Client SHALL return a descriptive error with the HTTP status code
5. THE Jenkins_API_Client SHALL use the same authentication and timeout configuration as existing methods
