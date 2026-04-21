# `fs`

## Table of contents

- [Summary](#summary)
- [Manual](#manual)
- [Subcommands](#subcommands)
  - [`scan`](#scan)
  - [`table`](#table)
- [Usage](#usage)

## Summary

A tool for scanning and analysing the file system.

## Manual

``` bash
$ inspex fs --help
NAME:
   inspex fs

USAGE:
   inspex fs [command options]

DESCRIPTION:
   File System tool

COMMANDS:
   scan     
   table    
   help, h  Shows a list of commands or help for one command

OPTIONS:
   --help, -h  show help
```

## Subcommands

### `scan`

``` bash
$ inspex fs scan --help
NAME:
   inspex fs scan

USAGE:
   inspex fs scan [command options]

DESCRIPTION:
   Scan a file or directory and output file system metadata as JSON or CSV

OPTIONS:
   --path value                               path to a file or directory to scan
   --output-format value                      output format (json or csv)
   --skip-path value [ --skip-path value ]    path to a file or directory to exclude from the scan (repeatable)
   --skip-regex value [ --skip-regex value ]  regular expression matched against file/directory base names to skip (repeatable)
   --skip-field value [ --skip-field value ]  exclude a metadata field from the output
   --recursive                                recursively scan directories (default: false)
   --help, -h                                 show help
```

### `table`

``` bash
$ inspex fs table --help
NAME:
   inspex fs table

USAGE:
   inspex fs table [command options]

DESCRIPTION:
   Render scanned file system metadata as an interactive HTML table

OPTIONS:
   --input-path value                                         path to the JSON file produced by the scan command
   --http                                                     serve the HTML table over HTTP on localhost (see --http-port) (default: false)
   --http-port value                                          port to listen on when --http is enabled (default: 8080)
   --display-field value [ --display-field value ]            field to include in the table (defaults to all fields present in input)
   --heat-map-field value [ --heat-map-field value ]          a field to use for temperature indication
   --exclude-date-field value [ --exclude-date-field value ]  date field to filter on when excluding rows (e.g. createdAt, modifiedAt)
   --exclude-date-value value [ --exclude-date-value value ]  date prefix to exclude rows by; accepts YYYY, YYYY-MM, or YYYY-MM-DD (e.g. 2024, 2024-03, 2024-03-15)
   --help, -h                                                 show help
```

## Usage

``` bash
inspex fs scan --path=. --output-format=json --recursive | inspex fs table --http
```
