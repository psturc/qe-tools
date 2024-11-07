package oci

// ArtifactScanner is used for pulling and extracting OCI artifact
// and scanning and storing files found in extracted content
type ArtifactScanner struct {
	config          ScannerConfig
	artifactDirPath string
	FilesPathMap    FilesPathMap
}

// ScannerConfig contains fields required
// for scanning files with ArtifactScanner
type ScannerConfig struct {
	OciArtifactReference string
	FileNameFilter       []string
}

// FilesPathMap - e.g. "e2e-test/e2e-report.xml": {Content: "<file-content>", Filename: "e2e-report.xml"}
type FilesPathMap map[FilePath]Artifact

// FilePath represents the full path of the file from the root directory of the extracted OCI artifact
type FilePath string

// Artifact stores the file name of the artifact and the content of the file
type Artifact struct {
	Content  string
	Filename string
}
