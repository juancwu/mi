# Mi CLI

This is a simple CLI to make encrypting/decrypting projects secrets that are safely stored in [Konbini](https://github.com/juancwu/mi)
among other features such as registering to Konbini, viewing all your bentos, etc... For all the available commands read [Command List](#command-list)

## Getting Started

### Method 1: Download from releases

```shell
# Change the variables as needed
curl -L -o mi https://github.com/juancwu/mi/releases/download/$VERSION/mi-$OS-$ARCH

# Make downloaded binary an executable
chmod +x mi

# Move executable to bin
sudo mv mi /usr/bin
```

### Method 2: Build Locally

You will need to have Go installed in your local machine.

```shell
# Clone the repo and cd into the repo
git clone https://github.com/juancwu/mi.git

# Build
make

# Move executable to bin
sudo mv mi /usr/bin
```

You can remove the repository if you want to.

## Command List

> Use `mi [command] --help` for more information about a specific command.

| Command                                     | Description                                                                                                                                                          |
| ------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `mi auth signup`                            | Use to register and unlock the features of Konbini.                                                                                                                  |
| `mi auth signin`                            | Use to sign into your Konbini account.                                                                                                                               |
| `mi auth verify-email <code>`               | Verify Konbini account email. Get code from an email sent to your account.                                                                                           |
| `mi auth resend-verification <email>`       | Request Konbini to resend a new email verification.                                                                                                                  |
| `mi auth reset-password`                    | Use to reset the password of your Konbini account.                                                                                                                   |
| `mi auth update-email`                      | Use to update the email of your Konbini account.                                                                                                                     |
| `mi auth delete-account`                    | Use to delete Konbini account.                                                                                                                                       |
| `mi bento order`                            | Request the data of a bento stored in Konbini. This **requires** a proper `.miconfig.yaml` present in the working directory.                                         |
| `mi bento prepare <bento-name>`             | Prepares a new empty bento which is stored in Konbini.                                                                                                               |
| `mi bento fill <path-to-env-file>`          | Fills a bento with the key value pairs in the given `.env` file. Any existing entries stored in Konbini will be erased and only the given `.env` values will remain. |
| `mi bento merge <path-to-env-file>`         | Merges the values of the given `.env` file with the ones stored in Konbini. Entries with the same `key` will not be overwritten unless `-force/-f` is provided.      |
| `mi bento add <key> [value]`                | Add a new ingridient to the bento. Optionally provide a value, or pass nothing to input value from STDIN.                                                            |
| `mi bento share <email> [--permissions=]`   | Share the bento with the user with `<email>`. Optionally, choose the permission(s) for the shared user.                                                              |
| `mi bento unshare <email> [--permissions=]` | Unshare the bento with the user with `<email>`. Optionally, choose the permission(s) to unshared.                                                                    |
| `mi ing rename <old-name> <new-name>`       | Renames an existing ingridient from a bento. Make sure you have the proper `.miconfig.yaml` and the private key for the bento.                                       |
| `mi ing reseason <old-name> <new-name>`     | Renames an existing ingridient from a bento. Make sure you have the proper `.miconfig.yaml` and the private key for the bento.                                       |
