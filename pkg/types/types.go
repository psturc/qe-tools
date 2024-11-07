package types

// Constants common across the whole project
const (
	ArtifactDirEnv    string = "ARTIFACT_DIR"
	GithubTokenEnv    string = "GITHUB_TOKEN" // #nosec G101
	ProwJobIDEnv      string = "PROW_JOB_ID"
	OciArtifactRefEnv string = "OCI_REF"

	ArtifactDirParamName             string = "artifact-dir"
	ProwJobIDParamName               string = "prow-job-id"
	OciArtifactRefParamName          string = "oci-ref"
	ClusterProvisionLogFileParamName string = "cluster-provision-log-name"
	E2ETestRunLogFileParamName       string = "e2e-log-name"
	JUnitFilenameParamName           string = "junit-report-name"
	OutputFilenameParamName          string = "output-file"

	JunitFilename string = `/(j?unit|e2e).*\.xml`
)

// CmdParameter represents an abstraction for viper parameters
type CmdParameter[T any] struct {
	Name         string
	Env          string
	DefaultValue T
	Value        T
	Usage        string
}
