# Design Document: Get Build Parameters

## Overview

This feature adds a `builds params` subcommand to jctl that retrieves and displays the parameters used when a specific build was triggered. The command will integrate with the existing Jenkins API client and follow the established patterns for command structure, authentication, and output formatting.

The implementation extends the existing `builds` command group with a new `params` subcommand that accepts a pipeline name and build number, retrieves the build information from Jenkins, extracts the parameters from the build's actions array, and displays them in the user's preferred format (text, JSON, or YAML).

## Architecture

### Component Overview

The feature consists of three main components:

1. **CLI Command Layer** (`cmd/jctl/builds.go`): Handles command parsing, validation, and output formatting
2. **API Client Layer** (`internal/client/client.go`): Communicates with Jenkins REST API to retrieve build information
3. **Data Model Layer** (`internal/client/models.go`): Defines the structure for build parameters

### Integration Points

- Extends the existing `buildsCmd` command group in `cmd/jctl/builds.go`
- Uses the existing `Client.GetBuildInfo()` method with enhanced parameter extraction
- Leverages existing authentication via `auth.Manager`
- Follows the same output formatting pattern as `builds list` command

### Data Flow

```
User Input (CLI) 
  → Command Validation 
  → API Client Request 
  → Jenkins REST API 
  → Parse Response 
  → Extract Parameters 
  → Format Output 
  → Display to User
```

## Components and Interfaces

### CLI Command Component

**Location**: `cmd/jctl/builds.go`

**New Command**:
```go
var buildsParamsCmd = &cobra.Command{
    Use:   "params <pipeline> <build-number>",
    Short: "Get parameters for a specific build",
    Long:  "Retrieve and display the parameters that were used when a specific build was triggered.",
    Example: "...",
    Args:  cobra.ExactArgs(2),
    RunE:  runBuildsParams,
}
```

**Key Functions**:
- `runBuildsParams(cmd *cobra.Command, args []string) error`: Main execution function
- `formatParamsOutput(params []client.Parameter, format string) error`: Formats output based on user preference
- `formatParamsText(params []client.Parameter) error`: Text table formatting
- `formatParamsJSON(params []client.Parameter) error`: JSON formatting
- `formatParamsYAML(params []client.Parameter) error`: YAML formatting

### API Client Enhancement

**Location**: `internal/client/client.go`

The existing `GetBuildInfo()` method will be enhanced to include parameter information in the API request tree parameter. The method already returns a `*Build` struct, but we need to ensure the API request includes the actions array with parameter information.

**Enhanced API Request**:
```go
path := fmt.Sprintf("%s/%d/api/json?tree=number,result,timestamp,duration,building,url,actions[parameters[name,value]]", jobPath, buildNumber)
```

**New Helper Function**:
```go
// ExtractParameters extracts parameter values from a build's actions array
func (b *Build) ExtractParameters() []Parameter
```

### Data Model

**Location**: `internal/client/models.go`

The `Parameter` struct already exists and is suitable for this feature:

```go
type Parameter struct {
    Name  string `json:"name"`
    Value string `json:"value"`
}
```

The `Build` struct needs to be updated to properly capture parameters from the actions array:

```go
type Build struct {
    Number     int         `json:"number"`
    URL        string      `json:"url"`
    Result     string      `json:"result"`
    Timestamp  int64       `json:"timestamp"`
    Duration   int64       `json:"duration"`
    Building   bool        `json:"building"`
    Actions    []BuildAction `json:"actions,omitempty"`
}

type BuildAction struct {
    Parameters []Parameter `json:"parameters,omitempty"`
}
```

## Data Models

### Build Action Structure

Jenkins stores build parameters within the `actions` array of a build. Each action can contain different types of information. The parameters are typically found in an action with `_class` of `hudson.model.ParametersAction`.

**Jenkins API Response Structure**:
```json
{
  "number": 42,
  "result": "SUCCESS",
  "actions": [
    {
      "_class": "hudson.model.ParametersAction",
      "parameters": [
        {
          "name": "ENVIRONMENT",
          "value": "production"
        },
        {
          "name": "VERSION",
          "value": "1.2.3"
        }
      ]
    },
    {
      "_class": "hudson.model.CauseAction",
      "causes": [...]
    }
  ]
}
```

### Parameter Extraction Logic

Since the `actions` array can contain multiple action types, we need to iterate through the array and extract parameters from the appropriate action(s). The extraction logic will:

1. Iterate through all actions in the build
2. Check each action for a `parameters` field
3. Collect all parameters from all actions that contain them
4. Return the consolidated list of parameters

### Output Formats

**Text Format** (default):
```
Parameters for build #42 of pipeline my-pipeline:

NAME            VALUE
----            -----
ENVIRONMENT     production
VERSION         1.2.3
```

**JSON Format**:
```json
[
  {
    "name": "ENVIRONMENT",
    "value": "production"
  },
  {
    "name": "VERSION",
    "value": "1.2.3"
  }
]
```

**YAML Format**:
```yaml
- name: ENVIRONMENT
  value: production
- name: VERSION
  value: "1.2.3"
```


## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Parameter Retrieval for Valid Inputs

*For any* valid pipeline name and build number, when the command is executed, the system should successfully retrieve the parameters from the Jenkins API and return them without error.

**Validates: Requirements 1.1**

### Property 2: Folder Path Support

*For any* pipeline name containing folder paths (in the format "folder/subfolder/pipeline"), the command should correctly URL-encode each path segment and successfully retrieve the build parameters.

**Validates: Requirements 1.5, 4.3**

### Property 3: Complete Parameter Display

*For any* set of parameters returned from the Jenkins API, all parameters should appear in the command output regardless of the output format chosen.

**Validates: Requirements 2.1**

### Property 4: Text Format Structure

*For any* set of parameters, when the output format is "text", the output should contain a table structure with aligned columns showing parameter names and values.

**Validates: Requirements 2.2**

### Property 5: Serialization Round-Trip

*For any* set of parameters, serializing to JSON or YAML and then deserializing should produce an equivalent set of parameters with the same names and values.

**Validates: Requirements 2.3, 2.4**

### Property 6: Output Format Consistency

*For any* valid output format value (text, json, yaml), the command should respect the global output format flag and produce output in the specified format.

**Validates: Requirements 2.5, 3.4**

### Property 7: Input Validation

*For any* command invocation, the command should accept exactly two arguments where the first is a string (pipeline name) and the second is a valid integer (build number), rejecting all other input combinations with appropriate error messages.

**Validates: Requirements 3.1, 3.3**

### Property 8: Global Flag Support

*For any* combination of global flags (profile, jenkins-url, output, verbose, timeout), the command should recognize and apply each flag correctly without conflicts.

**Validates: Requirements 3.4**

### Property 9: Error Response Structure

*For any* failed API request, the error returned should include a descriptive message and the HTTP status code from the Jenkins API.

**Validates: Requirements 4.4**

## Error Handling

### Error Categories

1. **Input Validation Errors**
   - Missing arguments (fewer than 2)
   - Too many arguments (more than 2)
   - Invalid build number (non-integer value)
   - Empty pipeline name

2. **API Errors**
   - Pipeline not found (404)
   - Build not found (404)
   - Authentication failure (401/403)
   - Network timeout
   - Connection refused
   - Invalid Jenkins URL

3. **Data Processing Errors**
   - Malformed JSON response from Jenkins
   - Missing expected fields in API response
   - Output formatting errors

### Error Messages

All error messages should follow the pattern established by existing jctl commands:

- **Pipeline not found**: `pipeline '<name>' not found\n\nDetails: No job with name '<name>' exists on the Jenkins server\nSuggestion: Use 'jctl pipelines list' to see available pipelines`

- **Build not found**: `build #<number> not found for pipeline '<name>'\n\nDetails: No build with number <number> exists for this pipeline\nSuggestion: Use 'jctl builds list <pipeline>' to see available builds`

- **Invalid build number**: `invalid build number: '<value>'\n\nDetails: Build number must be a positive integer\nExample: jctl builds params my-pipeline 42`

- **Missing arguments**: Display usage information with examples

- **No parameters**: `No parameters were used for build #<number> of pipeline <name>`

### Error Handling Strategy

1. **Validate inputs early**: Check argument count and build number format before making API calls
2. **Provide context**: Include the pipeline name and build number in error messages
3. **Suggest next steps**: Guide users toward resolution (e.g., list available builds)
4. **Preserve error details**: When wrapping errors, use `fmt.Errorf` with `%w` to maintain error chain
5. **Handle edge cases gracefully**: Empty parameter lists should display an informative message, not an error

## Testing Strategy

### Dual Testing Approach

This feature will be tested using both unit tests and property-based tests to ensure comprehensive coverage:

- **Unit tests**: Verify specific examples, edge cases, and error conditions
- **Property tests**: Verify universal properties across all inputs

### Unit Testing

Unit tests will focus on:

1. **Specific error cases**:
   - Pipeline not found (404 response)
   - Build not found (404 response)
   - Invalid build number format
   - Missing arguments

2. **Edge cases**:
   - Build with no parameters (empty actions array)
   - Build with multiple parameter actions
   - Parameters with special characters in names/values
   - Very long parameter values

3. **Integration points**:
   - Command registration with cobra
   - Authentication flow
   - API client method invocation

### Property-Based Testing

Property-based tests will use a Go property testing library (such as `gopter` or `rapid`) and will run a minimum of 100 iterations per test. Each test will be tagged with a comment referencing the design property.

**Property Test Configuration**:
- Library: `gopter` (Go property testing library)
- Iterations: 100 minimum per test
- Tag format: `// Feature: get-build-params, Property {number}: {property_text}`

**Property Tests to Implement**:

1. **Property 1 Test**: Generate random valid pipeline names and build numbers, mock successful API responses with random parameters, verify command succeeds
   - Tag: `// Feature: get-build-params, Property 1: Parameter Retrieval for Valid Inputs`

2. **Property 2 Test**: Generate random folder paths with special characters, verify URL encoding is correct and API calls succeed
   - Tag: `// Feature: get-build-params, Property 2: Folder Path Support`

3. **Property 3 Test**: Generate random parameter sets, verify all parameters appear in output
   - Tag: `// Feature: get-build-params, Property 3: Complete Parameter Display`

4. **Property 4 Test**: Generate random parameter sets, verify text output contains table structure
   - Tag: `// Feature: get-build-params, Property 4: Text Format Structure`

5. **Property 5 Test**: Generate random parameter sets, serialize to JSON/YAML, deserialize, verify equivalence
   - Tag: `// Feature: get-build-params, Property 5: Serialization Round-Trip`

6. **Property 6 Test**: Generate random format values, verify output matches requested format
   - Tag: `// Feature: get-build-params, Property 6: Output Format Consistency`

7. **Property 7 Test**: Generate random argument combinations, verify only exactly 2 arguments with valid types are accepted
   - Tag: `// Feature: get-build-params, Property 7: Input Validation`

8. **Property 8 Test**: Generate random flag combinations, verify all are recognized and applied
   - Tag: `// Feature: get-build-params, Property 8: Global Flag Support`

9. **Property 9 Test**: Generate random HTTP error codes, verify error messages include status codes
   - Tag: `// Feature: get-build-params, Property 9: Error Response Structure`

### Test Data Generation

For property-based tests, generators will create:
- Pipeline names: alphanumeric strings, with and without folder paths
- Build numbers: positive integers
- Parameters: arrays of name-value pairs with various string content
- Folder paths: nested paths with special characters requiring URL encoding
- HTTP status codes: 400, 401, 403, 404, 500, 502, 503

### Mocking Strategy

Tests will mock the Jenkins API client to avoid requiring a live Jenkins instance:
- Use interfaces to enable mock implementations
- Mock successful responses with generated parameter data
- Mock error responses with various HTTP status codes
- Verify API requests include correct tree parameters for fetching parameter data
