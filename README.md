# fifosplit

```bash:
mkfifo hoge.pipe
IN=hoge.pipe PATHFMT=hoge.%Y%m%d%H%M.log PERIOD=1m ./fifosplit
```
