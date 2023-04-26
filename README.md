# RollingF [![GoDoc](https://godoc.org/github.com/ignorantshr/rollingf?status.svg)](https://godoc.org/github.com/ignorantshr/rollingf)

RollingF is an IO tool that allows you to control the rules for rolling update files.

## Customize

Using RollingF you can customize four rules when rolling to update a file:
  1. Checker. Checker decides wheter trigger the rolling when write. For example, rolling the file every day or when the file size reaches 1M.
  1. Matcher. Matcher decides which files to be further processed. For example, app.log app.log.1 app.log.2 ...
  1. Filter. Filter filters files that you don't want to process in the processor. For example, some files that are too old, remove them.
  1. Processor. Processor processes the filtered files. For example, renaming the older files, rolling the new file.

```
+------------------+             +---------------+             +---------------+
|                  |             |               |             |               |
|  Write to file   +------------->    Checker    +------------->    rolling?   +----------N---------------+
|                  |             |               |             |               |                          |
+------------------+             +---------------+             +-------+-------+                          |
                                                                       |                                  |
                                                                       |                                  |
                                                                       Y                                  |
                                                                       |                                  |
                                                                       |                                  |
                                                                       |                      +-----------v-------------+
+------------------+                                           +-------v-------+              |                         |
|                  |                                           |               |              |                         |
|     Filter       <------------------some files---------------+    Mather     |              |  write content to file  |
|                  |                                           |               |              |                         |
+--------+---------+                                           +---------------+              |                         |
         |                                                                                    +-----------^-------------+
         |                                                                                                |
         +--------------------------------+                                                               |
         |                                |                                                               |
         |                                |                                                               |
+--------v---------+             +--------v---------+          +---------------+                          |
|                  |             |                  |          |               |                          |
| filtered files   |             | remaining files  +---------->   Processor   +-----after processed------+
|                  |             |                  |          |               |
+------------------+             +------------------+          +---------------+
```

## Default components

All the components has the default implementions.

- Checker
  - `IntervalChecker` checks whether a file should be rolled at regular intervals. If interval <= 0, it will never roll.
  - `MaxSizeChecker` checks whether a file should be rolled when its size exceeds maxSize.
- Matcher
  - `DefaultMatcher` matches the simple file names. eg. app.log app.log.1 app.log.2 ...
  - `CompressMatcher` matches the compressed file names. eg. app.log app.log.1.gz app.log.2.gz ...
- Filter
  - `MaxSizeFilter` filter files by size.
  - `MaxAgeFilter` filter files by age.
- Processor
  - `DefaultProcessor` renames the files, increase the tail number of the file name.
  - `Compressor` compress the files.

## Usage

### Simple usage

```go
    rf := rollingf.New(rollingf.NewRollConf("/tmp/any_app/any_app.log", time.Hour, 1024*1024, 30*24*time.Hour, 5))
    if rf == nil {
      return
    }

    rf.Write([]byte("simple rollingf"))
```

compress

```go
    rf := rollingf.New(rollingf.NewRollConf("/tmp/any_app/any_app.log", time.Hour, 1024*1024, 30*24*time.Hour, 5)).WithOptions(rollingf.Compress(rollingf.Gzip))
    if rf == nil {
      return
    }

    rf.Write([]byte("simple rollingf"))
```

### Customized usage

```go
    rf := rollingf.NewC("/tmp/any_app/any_app.log")
    if rf == nil {
      return
    }
    rf = rf.
      WithChecker(rollingf.IntervalChecker(rollingf.DurOneDay)).
      WithChecker(rollingf.MaxSizeChecker(rollingf.SizeMB)). // 1M
      WithFilter(rollingf.MaxAgeFilter(30 * rollingf.DurOneDay)).
      WithFilter(rollingf.MaxBackupsFilter(5)).
      WithDefaultMatcher().
      WithDefaultProcessor()

    defer rf.Close()

    rf.Write([]byte("hello rollingf"))
```

### Integration with standard library's log

```go
    rf := rollingf.New(rollingf.NewRollConf("/tmp/any_app/any_app.log", time.Hour, 1024*1024, 30*24*time.Hour, 5))
    if rf == nil {
      return
    }
    log.SetOutput(rf)

    log.Println("stdlib rollingf")
```

### Integration with zap

```go
var Logger *zap.SugaredLogger

func init() {
    core := logCore()
    if core == nil {
      panic("init logger failed")
    }
    Logger = zap.New(core, zap.AddCaller()).Sugar()
}

func logCore() zapcore.Core {
    rf := rollingf.New(
      rollingf.NewRollConf("/tmp/any_app/any_app.log", time.Hour, 1024*1024, 30*24*time.Hour, 5),
    )
    if rf == nil {
      return nil
    }

    encoder := zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig())
    writers := zapcore.NewMultiWriteSyncer(zapcore.AddSync(rf), zapcore.AddSync(os.Stderr))
    level := zap.NewAtomicLevelAt(zap.DebugLevel)
    return zapcore.NewCore(encoder, writers, level)
}
```
