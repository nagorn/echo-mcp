package control

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	ContractRootSourceEnv              = "ECHO_MCP_CONTRACT_ROOT"
	ContractRootSourceWorkingDirectory = "working_directory"
)

type contractRoot struct {
	path       string
	configured bool
	source     string
}

type resolvedContractPath struct {
	readPath   string
	sourcePath string
}

func newContractRoot(path string, source string) (contractRoot, error) {
	trimmedPath := strings.TrimSpace(path)
	configured := trimmedPath != ""
	if !configured {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return contractRoot{}, &OperationError{
				Code:        "contract_root_invalid",
				Message:     "contract root could not be resolved",
				Diagnostics: []string{"Default contract root uses the Echo MCP process working directory, but the working directory could not be resolved."},
			}
		}
		trimmedPath = workingDirectory
		source = ContractRootSourceWorkingDirectory
	} else if strings.TrimSpace(source) == "" {
		source = ContractRootSourceEnv
	}

	absoluteRoot, err := filepath.Abs(trimmedPath)
	if err != nil {
		return contractRoot{}, invalidContractRootError()
	}
	resolvedRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return contractRoot{}, invalidContractRootError()
	}
	info, err := os.Stat(resolvedRoot)
	if err != nil || !info.IsDir() {
		return contractRoot{}, invalidContractRootError()
	}

	return contractRoot{
		path:       filepath.Clean(resolvedRoot),
		configured: configured,
		source:     source,
	}, nil
}

func (r contractRoot) resolve(path string) (resolvedContractPath, error) {
	cleanedInput := filepath.Clean(strings.TrimSpace(path))
	if cleanedInput == "." {
		return resolvedContractPath{}, &OperationError{
			Code:        "missing_path",
			Message:     "path is required",
			Diagnostics: []string{"load_openapi_contract requires a local filesystem path."},
		}
	}

	candidate := cleanedInput
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(r.path, candidate)
	}
	absoluteCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return resolvedContractPath{}, contractPathNotAllowedError()
	}
	absoluteCandidate = filepath.Clean(absoluteCandidate)
	boundaryPath, err := evaluatePathForBoundary(absoluteCandidate)
	if err != nil {
		return resolvedContractPath{}, contractPathNotAllowedError()
	}
	if !pathWithinRoot(r.path, boundaryPath) {
		return resolvedContractPath{}, contractPathNotAllowedError()
	}

	sourcePath, err := filepath.Rel(r.path, boundaryPath)
	if err != nil {
		return resolvedContractPath{}, contractPathNotAllowedError()
	}
	sourcePath = filepath.Clean(sourcePath)
	return resolvedContractPath{
		readPath:   boundaryPath,
		sourcePath: sourcePath,
	}, nil
}

func evaluatePathForBoundary(path string) (string, error) {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(resolved), nil
	}

	parent := filepath.Dir(path)
	for {
		if resolvedParent, err := filepath.EvalSymlinks(parent); err == nil {
			relative, relErr := filepath.Rel(parent, path)
			if relErr != nil {
				return "", relErr
			}
			return filepath.Clean(filepath.Join(resolvedParent, relative)), nil
		}
		next := filepath.Dir(parent)
		if next == parent {
			return filepath.Clean(path), nil
		}
		parent = next
	}
}

func pathWithinRoot(root string, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if relative == "." {
		return true
	}
	if filepath.IsAbs(relative) || relative == ".." {
		return false
	}
	return !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func invalidContractRootError() *OperationError {
	return &OperationError{
		Code:        "contract_root_invalid",
		Message:     "contract root could not be resolved",
		Diagnostics: []string{"Contract root must resolve to an existing local directory."},
	}
}

func contractPathNotAllowedError() *OperationError {
	return &OperationError{
		Code:        "contract_path_not_allowed",
		Message:     "contract path is outside the allowed contract root",
		Diagnostics: []string{"OpenAPI contract paths must resolve under the configured contract root."},
	}
}
