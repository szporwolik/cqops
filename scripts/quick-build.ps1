$VERSION = Get-Content VERSION
$COMMIT = try { git rev-parse --short HEAD 2>$null } catch { "" }

# Regenerate Windows .syso resources if go-winres is available
if (Get-Command go-winres -ErrorAction SilentlyContinue) {
    Push-Location $PSScriptRoot\..\winres
    go-winres make --product-version $VERSION --file-version $VERSION --in winres.json --out ../cmd/cqops/rsrc
    Pop-Location
}

go build -ldflags "-s -w -X github.com/szporwolik/cqops/internal/version.Version=$VERSION -X github.com/szporwolik/cqops/internal/version.Commit=$COMMIT" -o build\cqops.exe ./cmd/cqops/
