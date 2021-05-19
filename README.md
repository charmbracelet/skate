# Skate

A personal key value store. 🛼

# What is it?

Skate is a key value store that you can use to save and retrieve valuable bits
of data. It's fully encrypted, backed up to the cloud (that you can self-host
if you want) and linkable to any machine you use.

## Examples

### Keep secrets out of your .bashrc

```
skate set gh_token GITHUB_TOKEN
echo 'export GITHUB_TOKEN=$(skate get gh_token)' >> ~/.bashrc
```

### Save some passwords

```
skate set github@password.db PASSWORD
skate get github@password.db
```

### Easily store data in scripts

```
#!/bin/bash

./skate set "$(date)@bookmarks.db" $1

./skate keys @bookmarks.db | while read b; do
  echo "$b $(./skate get "$b@bookmarks.db")"
done
```

# Installation

Use your fave package manager:

```
```

# Self-hosting

Skate is backed by the Charm Cloud. By default it will use the Charm hosted
cloud, but if you want to self-host you can download
[charm](https://github.com/charmbracelet/charm) and run you own cloud with
`charm serve`. Then set the `CHARM_HOST` environment variable to the hostname
of your Charm Cloud server.

# Developers

Skate is built on [charm/kv](https://github.com/charmbracelet/charm/kv). If
you'd like to build a tool that includes a user key value store, be sure to
check it out.

# License

[MIT](https://github.com/charmbracelet/skate/raw/master/LICENSE)

<a href="https://charm.sh/"><img alt="The Charm logo" src="https://stuff.charm.sh/charm-badge-unrounded.jpg" width="400"></a>

Charm热爱开源 • Charm loves open source
