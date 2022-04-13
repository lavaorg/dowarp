# dowarp
Examples of the Warp framework

## Example

to easily test the Warp library try the following

### run a warp service

The following sample service exports a few objects.

```
go run github.com/lavaorg/dowarp/cmd/w9@latest -d 1
```
the `-d 1` flag is optional and shows the __warp__ messages.

a `-d 3` option will dump the contents of the __warp__ packets as well.

### run a warp client

To access the serivce the following command line tool can be used to perform
generic access to objects. Use `-help` to get a list of browsing commands.

```
go run github.com/lavaorg/dowarp/cmd/warp@latest ctl /ctl memstats
```
