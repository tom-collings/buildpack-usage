# Buildpack Usage CLI Plugin
The plugin can be used to list of the apps that use the given build pack from
the CLI.
### Complilation

```bash
go get github.com/ecsteam/buildpack-usage
cd $GOPATH/src/github.com/ecsteam/buildpack-usage

GOOS=darwin go build -o buildpack-usage-plugin-macosx
GOOS=linux go build -o buildpack-usage-plugin-linux
GOOS=windows go build -o buildpack-usage-plugin-windows.exe
```
### Installation
```bash
cf install-plugin -r CF-Community "buildpack-usage"
```

### Usage
```
$ cf buildpack-usage [-b ${buildpack-name}]
```

If `${buildpack-name}` is omitted, you will be prompted to choose from a list of
installed buildpacks.
