# Modules

Extract any modules into this folder.

* Modules should be named uniquely, in a manner that identifies their purpose.
* Modules should be inside of a subfolder of `modules`, named after their package.
  * Example: `modules/birds/` would contain the `birds` module/package.

* Module folders should container a `datafiles` folder that contains any datafiles needed.
  * Files within `datafiles` will be treated as though located within the actual `_datafiles`
  * These files are read-only.

## Things Modules can do:

* Access core GoMud code.
* Listen for, handle and/or cancel events (See `modules/auctions`)
  * For example, run custom code every `NewRound{}` event, or do something whenever a `LevelUp{}` event is fired.
* Handle Telnet IAC commands (See `modules/gmcp`)
* Add a handler for new connections (See `modules/gmcp`)
* Add web pages to default web site (See `modules/leaderboards`)
  * Web page template with custom data
  * (optional) Add navigation links
  * (optional) provide any downloadable/linkable assets (images, files, etc)
* Add/Over-write existing template files (See `modules/auctions`)
* Add/Over-write help files (See `modules/auctions`)
* Add/Over-write user or mob commands  (See `modules/auctions`)
* Add functions for scripting (See `modules/follow`)
* Save/Load their own data (See `modules/leaderboards`)
* Track their own config values (See `modules/leaderboards`)
* Modify help menu items, command aliases, help aliases  (See `modules/leaderboards`)

# Examples

## Basic user command function

* time/time.go
* time/files/*

## User command with maintained state and save/loading of data

* leaderboards/leaderboards.go
* leaderboards/files/*
