# Teryx: A Fossil SCM Workflow Companion

**Teryx** (short for *Archaeopteryx*) is a small, opinionated command-line tool for Debian Linux that streamlines common workflows for the [Fossil SCM](https://fossil-scm.org/). It's designed for developers who love Fossil's all-in-one approach but want to accelerate the initial setup, transfer, and cloning of repositories.

## Why Teryx?

Fossil SCM is a powerful, self-contained version control system that excels at many things, particularly its efficient handling of binary files. This makes it an outstanding choice for projects that go beyond just source code, such as versioning assets for game development, design projects, or as I use it, audio files for an audiobook production.

Teryx was born from a desire to make the most common Fossil workflows even faster. It doesn't replace the `fossil` binary; it complements it by automating the multi-step processes of:

1.  **Initializing** a new repository and its first administrator user.
2.  **Cloning** a remote repository into a clean, organized local directory structure.
3.  **Transferring** a repository to a server and providing the exact permissions command needed to get it running with a web server.

Teryx handles the boilerplate so you can get back to what matters: your project.

## Features

* **Fast Init:** Create a new `.fossil` file, a corresponding checkout directory, and an admin user with a password of your choice in a single command.
* **Smart Cloning:** Clones a remote repository into a structured `$HOME/fossils/<hostname>/<path>` directory, mimicking the remote structure for local organization.
* **Guided Transfer:** Uses `scp` (with an `sftp` fallback) to push your repository to a server, then prints the exact `ssh` command you need to run to set the correct ownership and permissions for web server access.
* **Sensible Defaults:** Automatically uses your system username but lets you override it. Automatically appends the `.fossil` extension to new repos.

## Installation (from source)

Teryx is a single Go binary with one dependency. Building it is simple.

1.  **Install Go:** Ensure you have a recent version of Go installed on your Debian system.
2.  **Get Dependencies:** Create a project directory and run the following commands:
    ```bash
    go mod init teryx
    go get [github.com/spf13/cobra@latest](https://github.com/spf13/cobra@latest)
    ```
3.  **Build:**
    ```bash
    go build -o teryx .
    ```
4.  **Install (Optional):** Move the resulting `teryx` binary to a location in your system's PATH.
    ```bash
    sudo mv teryx /usr/local/bin/
    ```

## Usage

### `teryx init`

Initializes a new repository and a clean checkout directory for it.

```
teryx init <repository-name> --password <your-password> [--user <admin-user>]
```

* **`<repository-name>`:** The name for your project. `.fossil` will be appended automatically if you omit it.
* **`--password, -p`:** (Required) The password for the new admin user.
* **`--user, -u`:** (Optional) The admin username. Defaults to the output of `whoami`.

**Example:**
```
# Creates tester.fossil and a ./tester/ checkout directory
teryx init tester -p "s3cureP@ssw0rd!"
```

### `teryx clone`

Clones a remote Fossil repository into a structured local directory.

```
teryx clone <fossil-url>
```

* **`<fossil-url>`:** The full HTTP/HTTPS URL to the repository. Teryx will automatically strip a trailing `/home` if it exists.

**Example:**
```
teryx clone [http://fossil.example.com/my-project/home](http://fossil.example.com/my-project/home)

# This will:
# 1. Create the directory ~/fossils/[fossil.example.com/](https://fossil.example.com/)
# 2. Clone the repo into ~/fossils/[fossil.example.com/my-project.fossil](https://fossil.example.com/my-project.fossil)
# 3. Create a checkout directory at ~/.fossils/[fossil.example.com/my-project/](https://fossil.example.com/my-project/)
# 4. Open the repository in the new checkout directory.
```

### `teryx transfer`

Transfers a local `.fossil` file to a remote server.

```
teryx transfer <repository-name> --destination <user@host:path> [--remote-user <web-user>]
```

* **`<repository-name>`:** The local `.fossil` file to transfer.
* **`--destination, -d`:** (Required) The `scp`-style destination (e.g., `user@myserver.com:/srv/fossil/`).
* **`--remote-user, -r`:** (Optional) The user/group of your web server on the remote host. Defaults to `www-data`.

**Example:**
```
# Transfer the file and get a permissions command tailored for the 'fossil' user
teryx transfer tester.fossil -d <username>@<server>:/var/lib/fossil -r fossil
```

---
*This tool was specified and implemented with the assistance of Google's Gemini.*

