# `glv`

## Table of contents

- [Summary](#summary)
- [Manual](#manual)
- [Subcommands](#subcommands)
  - [`parse`](#parse)
  - [`show`](#show)
- [Usage](#usage)

## Summary

A tool for scanning and analysing the git log.

## Manual

``` bash
$ inspex glv --help
NAME:
   inspex glv

USAGE:
   inspex glv [command options]

COMMANDS:
   parse    
   show     
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --help, -h  show help
```

## Subcommands

### `parse`

``` bash
$ inspex glv parse --help
NAME:
   inspex glv parse

USAGE:
   inspex glv parse [command options]

OPTIONS:
   --path value  path to the git repository (defaults to current directory)
   --help, -h    show help
```

### `show`

``` bash
$ inspex glv show --help
NAME:
   inspex glv show

USAGE:
   inspex glv show [command options]

OPTIONS:
   --input-path value                             path to the JSON file produced by the parse command
   --http                                         serve the HTML table over HTTP on localhost (see --http-port) (default: false)
   --http-port value                              port to listen on when --http is enabled (default: 8080)
   --include-date value [ --include-date value ]  only show commits on this date (YYYY-MM-DD), repeatable
   --exclude-date value [ --exclude-date value ]  exclude all commits on this date (YYYY-MM-DD), repeatable
   --include-time-from value                      only show commits at or after this time of day (HH:MM)
   --include-time-to value                        only show commits at or before this time of day (HH:MM)
   --help, -h                                     show help
```

## Usage

``` bash
inspex glv parse | inspex glv show --http
```
