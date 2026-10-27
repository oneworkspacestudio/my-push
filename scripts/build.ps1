param([string]$Go='go',[string]$Out='I:/Projects/tooling/my-push',[string]$Version='0.1.0-dev')
$ErrorActionPreference='Stop'
$root=Split-Path $PSScriptRoot -Parent
New-Item -ItemType Directory -Force $Out | Out-Null
if($env:GOTMPDIR){New-Item -ItemType Directory -Force $env:GOTMPDIR | Out-Null}
$env:CGO_ENABLED='0';$env:GOARCH='amd64';$env:GOMAXPROCS='2'
Push-Location $root
try {
 $env:GOOS='windows'
 & $Go run github.com/akavel/rsrc@v0.10.2 -ico (Join-Path $root 'branding/my_push.ico') -o (Join-Path $root 'cmd/my-push/rsrc_windows_amd64.syso')
 if($LASTEXITCODE){throw 'My Push Windows icon resource failed'}
 foreach($os in @('windows','linux')){
  $env:GOOS=$os
  $name=if($os -eq 'windows'){'MyPush.exe'}else{'MyPush-linux-amd64'}
  & $Go build -buildvcs=false -trimpath -ldflags "-s -w -X main.version=$Version" -o (Join-Path $Out $name) ./cmd/my-push
  if($LASTEXITCODE){throw "My Push $os build failed"}
 }
 foreach($name in @('LICENSE','NOTICE','THIRD-PARTY-LICENSES.txt','compatibility.json')){Copy-Item -LiteralPath (Join-Path $root $name) -Destination (Join-Path $Out $name) -Force}
 $compat=Get-Content -LiteralPath (Join-Path $Out 'compatibility.json') -Raw|ConvertFrom-Json
 $compat.version=$Version
 [IO.File]::WriteAllText((Join-Path $Out 'compatibility.json'),($compat|ConvertTo-Json -Depth 8))
 $app=@{id='push';name='My Push';description='Private notification delivery for your Workspace apps';version=$Version;kind='process';package='push';executable='MyPush.exe';port=9019;route='/push';stripPrefix=$true;health='/healthz';depends=@('accounts');oidc=$true;callback='/push/auth/callback';args=@('--data','${DATA}/push','--listen','127.0.0.1:${PORT}','--origin','${ORIGIN}/push','--no-browser');source="https://github.com/oneworkspacestudio/my-push/releases/tag/v$Version"}
 $packages=[ordered]@{}
 foreach($row in @(@('push','MyPush.exe','MyPush.exe'),@('push-LICENSE','LICENSE','licenses/MyPush-LICENSE'),@('push-NOTICE','NOTICE','licenses/MyPush-NOTICE'),@('push-THIRD-PARTY-LICENSES.txt','THIRD-PARTY-LICENSES.txt','licenses/MyPush-THIRD-PARTY-LICENSES.txt'))){$file=Join-Path $Out $row[1];$packages[$row[0]]=@{file=$row[2];asset=$row[1];size=(Get-Item -LiteralPath $file).Length;sha256=(Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash.ToLowerInvariant()}}
 $manifest=@{schema=2;integration=3;repository='oneworkspacestudio/my-push';tag="v$Version";minimumManager='0.3.0-preview.94';requiredCapabilities=@('self-hosted-push-v1');apps=@($app);packages=$packages}
 [IO.File]::WriteAllText((Join-Path $Out 'workspace-update.json'),($manifest|ConvertTo-Json -Depth 8))
 Copy-Item -LiteralPath (Join-Path $Out 'workspace-update.json') -Destination (Join-Path $Out 'workspace-app-update.json') -Force
 Get-FileHash -LiteralPath (Join-Path $Out 'MyPush.exe'),(Join-Path $Out 'MyPush-linux-amd64') -Algorithm SHA256
}finally{Pop-Location}
