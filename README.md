# Konbini CLI

This is a simple CLI to make encrypting/decrypting projects secrets that are safely stored in [Konbini](https://github.com/juancwu/konbini)
among other features such as registering to Konbini, viewing all your bentos, etc... For all the available commands read [Command List](#command-list)

## Getting Started

### Method 1: Download from releases

```shell
# Change the variables as needed
curl -L -o konbini https://github.com/juancwu/konbini-cli/releases/download/$VERSION/konbini-$OS-$ARCH

# Make downloaded binary an executable
chmod +x konbini

# Move executable to bin
sudo mv konbini /usr/bin
```

### Method 2: Build Locally

You will need to have Go installed in your local machine.
 
```shell
# Clone the repo and cd into the repo
git clone git@github.com:juancwu/konbini-cli.git

# Build
go build -o konbini .

# Move executable to bin
sudo mv konbini /usr/bin
```

You can remove the repository if you want to.

## Command List

> Use `konbin [command] --help` for more information about a specific command

| Command | Description |
| ------- | ----------- |
| `konbini get membership` | Use to register and unlock the features of Konbini. |
| `konbini get bento` | Get a bento based on the `.konbini.yaml` that is generated when a new bento is cooked with `konbini cook bento [name]`. The public/private keys must be accessible. |
| `konbini cook bento` | Cook a new bento to store in Konbini. This will read the `.env` file present in the `cwd`. New RSA keys will be generated as well as the `.konbini.yaml` configuration file. |
