#!/bin/bash

# EdgeStream CI Configuration Validation Script
# This script validates GitHub Actions workflows and configurations

set -e

echo "🔍 Validating EdgeStream CI Configuration..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    case $status in
        "success")
            echo -e "${GREEN}✅ $message${NC}"
            ;;
        "warning")
            echo -e "${YELLOW}⚠️  $message${NC}"
            ;;
        "error")
            echo -e "${RED}❌ $message${NC}"
            ;;
        "info")
            echo -e "${BLUE}ℹ️  $message${NC}"
            ;;
    esac
}

# Check if required tools are installed
check_tools() {
    print_status "info" "Checking required tools..."
    
    local tools=("go" "docker" "docker-compose" "git")
    local missing_tools=()
    
    for tool in "${tools[@]}"; do
        if command -v "$tool" &> /dev/null; then
            print_status "success" "$tool is installed"
        else
            missing_tools+=("$tool")
            print_status "error" "$tool is not installed"
        fi
    done
    
    if [ ${#missing_tools[@]} -gt 0 ]; then
        print_status "error" "Missing tools: ${missing_tools[*]}"
        return 1
    fi
}

# Validate GitHub Actions workflow files
validate_workflows() {
    print_status "info" "Validating GitHub Actions workflows..."
    
    local workflow_dir=".github/workflows"
    
    if [ ! -d "$workflow_dir" ]; then
        print_status "error" "GitHub workflows directory not found: $workflow_dir"
        return 1
    fi
    
    local workflows=("ci.yml" "code-quality.yml" "dependabot.yml")
    
    for workflow in "${workflows[@]}"; do
        local workflow_path="$workflow_dir/$workflow"
        if [ -f "$workflow_path" ]; then
            print_status "success" "Found workflow: $workflow"
            
            # Basic YAML syntax validation
            if command -v yq &> /dev/null; then
                if yq eval '.' "$workflow_path" > /dev/null 2>&1; then
                    print_status "success" "$workflow has valid YAML syntax"
                else
                    print_status "error" "$workflow has invalid YAML syntax"
                fi
            else
                print_status "warning" "yq not installed, skipping YAML validation"
            fi
        else
            print_status "error" "Missing workflow: $workflow"
        fi
    done
}

# Validate Dependabot configuration
validate_dependabot() {
    print_status "info" "Validating Dependabot configuration..."
    
    local dependabot_config=".github/dependabot.yml"
    
    if [ -f "$dependabot_config" ]; then
        print_status "success" "Found Dependabot configuration"
        
        # Check for required package ecosystems
        local ecosystems=("gomod" "github-actions" "docker")
        
        for ecosystem in "${ecosystems[@]}"; do
            if grep -q "package-ecosystem: $ecosystem" "$dependabot_config"; then
                print_status "success" "Dependabot configured for $ecosystem"
            else
                print_status "warning" "Dependabot not configured for $ecosystem"
            fi
        done
    else
        print_status "error" "Dependabot configuration not found"
    fi
}

# Validate golangci-lint configuration
validate_golangci() {
    print_status "info" "Validating golangci-lint configuration..."
    
    local golangci_config=".golangci.yml"
    
    if [ -f "$golangci_config" ]; then
        print_status "success" "Found golangci-lint configuration"
        
        # Check if golangci-lint can parse the config
        if command -v golangci-lint &> /dev/null; then
            if golangci-lint config verify 2>/dev/null; then
                print_status "success" "golangci-lint configuration is valid"
            else
                print_status "error" "golangci-lint configuration is invalid"
            fi
        else
            print_status "warning" "golangci-lint not installed, skipping validation"
        fi
        
        # Check for deprecated linters
        local deprecated_linters=("deadcode" "golint" "interfacer" "maligned" "scopelint" "structcheck" "varcheck")
        
        for linter in "${deprecated_linters[@]}"; do
            if grep -q "$linter" "$golangci_config"; then
                print_status "warning" "Deprecated linter found: $linter"
            fi
        done
    else
        print_status "error" "golangci-lint configuration not found"
    fi
}

# Validate Docker configuration
validate_docker() {
    print_status "info" "Validating Docker configuration..."
    
    local docker_files=("Dockerfile" "Dockerfile.dev" "docker-compose.yml" "docker-compose.dev.yml")
    
    for file in "${docker_files[@]}"; do
        if [ -f "$file" ]; then
            print_status "success" "Found Docker file: $file"
        else
            print_status "warning" "Missing Docker file: $file"
        fi
    done
    
    # Validate docker-compose files
    if command -v docker-compose &> /dev/null; then
        for compose_file in "docker-compose.yml" "docker-compose.dev.yml"; do
            if [ -f "$compose_file" ]; then
                if docker-compose -f "$compose_file" config > /dev/null 2>&1; then
                    print_status "success" "$compose_file is valid"
                else
                    print_status "error" "$compose_file is invalid"
                fi
            fi
        done
    fi
}

# Validate Go module
validate_go_module() {
    print_status "info" "Validating Go module..."
    
    if [ -f "go.mod" ]; then
        print_status "success" "Found go.mod"
        
        # Check Go version
        local go_version
        go_version=$(go version | awk '{print $3}' | sed 's/go//')
        print_status "info" "Go version: $go_version"
        
        # Validate go.mod
        if go mod verify > /dev/null 2>&1; then
            print_status "success" "go.mod is valid"
        else
            print_status "error" "go.mod validation failed"
        fi
        
        # Check for tidy
        if go mod tidy -diff > /dev/null 2>&1; then
            print_status "success" "go.mod is tidy"
        else
            print_status "warning" "go.mod needs tidying"
        fi
    else
        print_status "error" "go.mod not found"
    fi
}

# Validate project structure
validate_structure() {
    print_status "info" "Validating project structure..."
    
    local required_dirs=("cmd" "internal" "pkg" "config" "scripts" "monitoring")
    local optional_dirs=("docs" "examples" "test" "build")
    
    for dir in "${required_dirs[@]}"; do
        if [ -d "$dir" ]; then
            print_status "success" "Found required directory: $dir"
        else
            print_status "error" "Missing required directory: $dir"
        fi
    done
    
    for dir in "${optional_dirs[@]}"; do
        if [ -d "$dir" ]; then
            print_status "success" "Found optional directory: $dir"
        else
            print_status "info" "Optional directory not found: $dir"
        fi
    done
    
    # Check for important files
    local important_files=("README.md" "LICENSE" ".gitignore")
    
    for file in "${important_files[@]}"; do
        if [ -f "$file" ]; then
            print_status "success" "Found important file: $file"
        else
            print_status "warning" "Missing important file: $file"
        fi
    done
}

# Main validation function
main() {
    echo "🚀 Starting EdgeStream CI validation..."
    echo ""
    
    local exit_code=0
    
    # Run all validations
    check_tools || exit_code=1
    echo ""
    
    validate_workflows || exit_code=1
    echo ""
    
    validate_dependabot || exit_code=1
    echo ""
    
    validate_golangci || exit_code=1
    echo ""
    
    validate_docker || exit_code=1
    echo ""
    
    validate_go_module || exit_code=1
    echo ""
    
    validate_structure || exit_code=1
    echo ""
    
    # Summary
    if [ $exit_code -eq 0 ]; then
        print_status "success" "All validations passed! 🎉"
    else
        print_status "error" "Some validations failed. Please check the output above."
    fi
    
    echo ""
    echo "📋 Validation Summary:"
    echo "   - GitHub Actions workflows"
    echo "   - Dependabot configuration"
    echo "   - golangci-lint configuration"
    echo "   - Docker configuration"
    echo "   - Go module validation"
    echo "   - Project structure"
    
    return $exit_code
}

# Run main function
main "$@"