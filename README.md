# Buildpack Usage CLI Plugin
The plugin can be used to list of the apps that use the given build pack from the CLI.
### Complilation

```bash
go get github.com/ecsteam/buildpack-usage
cd $GOPATH/src/github.com/ecsteam/buildpack-usage

GOOS=darwin go build -o buildpack-usage-plugin-osx
GOOS=linux go build -o buildpack-usage-plugin-linux
GOOS=windows go build -o buildpack-usage-plugin-windows.exe
```
### Installation
```bash
cf install-plugin /path/to/buildpack-usage-plugin-<os build>
```
### Usage
```
cf buildpack-usage [-b ${name_of_buildpack}]
```
