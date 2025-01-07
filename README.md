# GPX Out and Back Fixer

> This has only been tested with Garmin GPX files, your results may vary. But I was able to import it into [gpx.studio](https://gpx.studio/) and edit it as well.

## GPX Issue

GPX has an inherent issue with out and back routes because it can't differentiate between the out leg or the return leg. So if you have waypoints on your route its up to the client app to determine how to show them to the user. Either showing the distance to them on both legs, the out leg, or the return leg. Garmin courses seems to only show them on the return leg which kind of sucks if you want to know the distance for both.

In Garmin when adding a waypoint to the course it will always add it as the return leg of the route with no way to add it on the out leg.

## Program Description

This will take a GPX file that was created in Garmin using an out and back method, duplicate all of the track points, offset them by 3 meters, and add a waypoint called `TURN AROUND` to the file. Doing this will allow you to see 2 distinct lines in your course editor. You can now add waypoints to the route and put them on the specific leg. I labeled mine like `Aid 1 - Out` & `Aid 1 - Rtn` to help tell them apart on the watch.
