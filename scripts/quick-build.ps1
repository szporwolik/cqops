$VERSION = Get-Content VERSION
$COMMIT = try { git rev-parse --short HEAD 2>$null } catch { "" }

# Ensure Go is on PATH (VS Code task shells can start before Go was installed/registered).
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    foreach ($p in @("$env:ProgramFiles\Go\bin", "${env:ProgramFiles(x86)}\Go\bin", "$env:LOCALAPPDATA\Programs\Go\bin", "$env:USERPROFILE\go\bin", "$env:USERPROFILE\scoop\apps\go\current\bin")) {
        if (Test-Path (Join-Path $p 'go.exe')) { $env:Path = "$p;$env:Path"; break }
    }
}

# Regenerate Windows .syso resources if go-winres is available
if (Get-Command go-winres -ErrorAction SilentlyContinue) {
    Push-Location $PSScriptRoot\..\winres
    go-winres make --product-version $VERSION --file-version $VERSION --in winres.json --out ../cmd/cqops/rsrc
    Pop-Location
}

go build -ldflags "-s -w -X github.com/szporwolik/cqops/internal/version.Version=$VERSION -X github.com/szporwolik/cqops/internal/version.Commit=$COMMIT" -o build\cqops.exe ./cmd/cqops/
