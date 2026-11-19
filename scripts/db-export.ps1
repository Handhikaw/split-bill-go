# SplitKita - export database lokal (Docker) untuk dipindah ke Coolify.
#
# Menghasilkan:
#   <prefix>-<timestamp>.sql          plain SQL, portable: siap dipakai sebagai
#                                     Coolify initialization script
#   <prefix>-<timestamp>.dump         custom format, untuk pg_restore (fallback)
#   <prefix>-<timestamp>-verify.txt   row count + cek objek migration
#   <prefix>-<timestamp>.sha256       checksum kedua file dump
#   <prefix>-<timestamp>-manifest.txt ringkasan
#
# Contoh:
#   .\scripts\db-export.ps1
#   .\scripts\db-export.ps1 -Container postgres -DbUser postgres -DbName postgres
#
# Password tidak ditulis ke file. Kalau database lokal butuh password:
#   $env:PGPASSWORD = "..."   lalu jalankan script ini.

[CmdletBinding()]
param(
    [string]$Container = "postgres",
    [string]$DbUser = "postgres",
    [string]$DbName = "postgres",
    [string]$DbPassword = $env:PGPASSWORD,
    [string]$OutputDir = "",
    [string]$Prefix = "splitkita"
)

$ErrorActionPreference = "Stop"

function Assert-LastExitCode {
    param([string]$Step)
    if ($LASTEXITCODE -ne 0) {
        throw "$Step gagal (exit code $LASTEXITCODE)."
    }
}

function Invoke-Psql {
    param(
        [string]$Container,
        [string]$DbUser,
        [string]$DbName,
        [string]$DbPassword,
        [string[]]$PsqlArgs
    )

    $dockerArgs = @("exec")
    if ($DbPassword) {
        $dockerArgs += @("-e", "PGPASSWORD=$DbPassword")
    }
    $dockerArgs += @($Container, "psql", "-U", $DbUser, "-d", $DbName)
    $dockerArgs += $PsqlArgs
    & docker @dockerArgs
    Assert-LastExitCode "psql"
}

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "docker tidak ditemukan di PATH."
}

if (-not $OutputDir) {
    $OutputDir = Join-Path (Split-Path -Parent $PSScriptRoot) "backups"
}
$OutputDir = [System.IO.Path]::GetFullPath($OutputDir)
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

# Pastikan container hidup.
$running = (& docker inspect --format "{{.State.Running}}" $Container 2>$null)
Assert-LastExitCode "docker inspect $Container"
if ($running.Trim() -ne "true") {
    throw "Container '$Container' tidak berjalan. Jalankan dulu: docker start $Container"
}

$serverVersion = (& docker exec $Container postgres --version)
Assert-LastExitCode "cek versi postgres"
$serverVersion = ($serverVersion -join " ").Trim()

$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$baseName = "$Prefix-$timestamp"
$sqlFile = Join-Path $OutputDir "$baseName.sql"
$dumpFile = Join-Path $OutputDir "$baseName.dump"
$verifyFile = Join-Path $OutputDir "$baseName-verify.txt"
$checksumFile = Join-Path $OutputDir "$baseName.sha256"
$manifestFile = Join-Path $OutputDir "$baseName-manifest.txt"

$containerSql = "/tmp/$baseName.sql"
$containerDump = "/tmp/$baseName.dump"
$containerVerify = "/tmp/$baseName-verify.sql"

$verifySource = Join-Path $PSScriptRoot "db-verify.sql"
if (-not (Test-Path $verifySource)) {
    throw "File db-verify.sql tidak ditemukan di $PSScriptRoot"
}

Write-Host "==> Export database '$DbName' dari container '$Container' ($serverVersion)"
Write-Host "==> Output: $OutputDir"

# 1) Verifikasi / row count sebelum dump.
& docker cp $verifySource "${Container}:$containerVerify"
Assert-LastExitCode "docker cp db-verify.sql"

$verifyOutput = New-Object System.Collections.Generic.List[string]
$verifyOutput.Add("# SplitKita verify - $baseName")
$verifyOutput.Add("# source: $Container / $DbName / $serverVersion")
$verifyOutput.Add("")

Invoke-Psql -Container $Container -DbUser $DbUser -DbName $DbName `
    -DbPassword $DbPassword -PsqlArgs @("-v", "ON_ERROR_STOP=1", "-f", $containerVerify) |
    ForEach-Object { $verifyOutput.Add($_) }

$verifyOutput | Out-File -FilePath $verifyFile -Encoding utf8

$tableRows = ($verifyOutput | Where-Object { $_ -match "^\s*\w+\s+\|\s+\d+\s*$" } | Measure-Object).Count
$totalRows = 0
foreach ($line in $verifyOutput) {
    if ($line -match "^\s*\w+\s+\|\s+(\d+)\s*$") {
        $totalRows += [int64]$Matches[1]
    }
}

# 2) Plain SQL portable untuk initialization script Coolify.
#    --no-owner / --no-privileges -> objek jadi milik user yang menjalankan script.
& docker exec -e "PGPASSWORD=$DbPassword" $Container `
    pg_dump -U $DbUser -d $DbName `
    --no-owner --no-privileges --no-tablespaces `
    -Fp -f $containerSql
Assert-LastExitCode "pg_dump plain SQL"

& docker cp "${Container}:$containerSql" $sqlFile
Assert-LastExitCode "docker cp plain SQL"

# 3) Custom format untuk pg_restore (fallback / arsip).
& docker exec -e "PGPASSWORD=$DbPassword" $Container `
    pg_dump -U $DbUser -d $DbName `
    --no-owner --no-privileges --no-tablespaces `
    -Fc -f $containerDump
Assert-LastExitCode "pg_dump custom format"

& docker cp "${Container}:$containerDump" $dumpFile
Assert-LastExitCode "docker cp custom dump"

# 4) Cek dump custom bisa dibaca.
$tocLines = (& docker exec $Container pg_restore --list $containerDump | Measure-Object -Line).Lines
Assert-LastExitCode "pg_restore --list"

# 5) Bersihkan file sementara di container.
& docker exec $Container rm -f $containerSql $containerDump $containerVerify | Out-Null

# 6) Checksum.
$hashes = Get-FileHash -Algorithm SHA256 -Path $sqlFile, $dumpFile
$hashes | ForEach-Object { "$($_.Hash.ToLower())  $(Split-Path -Leaf $_.Path)" } |
    Out-File -FilePath $checksumFile -Encoding ascii
Assert-LastExitCode "checksum"

# 7) Manifest.
$sqlInfo = Get-Item $sqlFile
$dumpInfo = Get-Item $dumpFile
$manifest = @(
    "# SplitKita database export",
    "",
    "- created_at : $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss K')",
    "- source     : container=$Container db=$DbName user=$DbUser",
    "- postgres   : $serverVersion",
    "- tables     : $tableRows",
    "- total_rows : $totalRows (mengikuti urutan verify; angka per tabel ada di file verify)",
    "- plain_sql  : $($sqlInfo.Name) ($([math]::Round($sqlInfo.Length / 1KB, 1)) KB) - init script Coolify",
    "- custom     : $($dumpInfo.Name) ($([math]::Round($dumpInfo.Length / 1KB, 1)) KB) - pg_restore",
    "- toc_entries: $tocLines",
    "",
    "Restore plain SQL (fresh DB):",
    "  psql -U <user> -d <db> -v ON_ERROR_STOP=1 -f $($sqlInfo.Name)",
    "",
    "Restore custom dump:",
    "  pg_restore -U <user> -d <db> --clean --if-exists --no-owner --no-privileges $($dumpInfo.Name)",
    "",
    "Jangan commit folder backups/ - berisi data pengguna."
)
$manifest | Out-File -FilePath $manifestFile -Encoding utf8

Write-Host ""
Write-Host "Selesai:"
Write-Host "  SQL    : $sqlFile"
Write-Host "  DUMP   : $dumpFile"
Write-Host "  VERIFY : $verifyFile"
Write-Host "  SHA256 : $checksumFile"
Write-Host "  MANIFEST: $manifestFile"
Write-Host ""
Write-Host "Langkah berikutnya: upload '$($sqlInfo.Name)' sebagai initialization script"
Write-Host "di resource PostgreSQL Coolify (DB harus masih fresh / volume kosong)."
