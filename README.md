
# go-dns-update

[![GPLv3 License](https://img.shields.io/badge/License-GPL%20v3-yellow.svg)](https://choosealicense.com/licenses/gpl-3.0/)

Automatic DNS record updater for Cloudflare written in Go.


## Run Locally

Clone the project

```bash
  git clone https://github.com/TheSilverBulet/go-dns-update.git
```

Go to the project directory

```bash
  cd my-project
```

Build binary

```bash
  go build main.go
```

Run program

```bash
  ./main -flag1=a -flag2=b
```

Program help

```bash
  ./main -h
```


## FAQ

#### Why?

Wanted to challenge myself to build something useful in Go.

#### Why not DDClient?

DDClient works great and does exactly what it was meant to while also being a mature project. This program attempts to solve a similar problem that DDClient solves, except only for a specific case (Cloudflare), and this program doesn't include any cron or timing element so it needs to be setup separately.

