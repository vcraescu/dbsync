# DBSync

Sync two mysql databases.

### Config file

Create an YAML file inside current directory or `~/.dbsync.yml`.

E.g:

```
servers:
  master:
    username: "mysql_username"
    password: "mysql_password"
    host: "localhost"
    port: 3306
    schema: "master_db"

  slave:
    username: "mysql_username"
    password: "mysql_password"
    host: "localhost"
    port: 3306
    schema: "slave_db"
    ssh:
      user: "my_ssh_user"
      host: "example.com"
      port: 22
      key: "~/.ssh/your_pk_file" # if omitted, ssh agent keys is used
```
