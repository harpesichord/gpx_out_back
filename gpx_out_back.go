package main

import (
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"strings"
)

// GPX structures
type GPX struct {
	XMLName   xml.Name   `xml:"gpx"`
	Creator   string     `xml:"creator,attr"`
	Version   string     `xml:"version,attr"`
	XMLNS     string     `xml:"xmlns,attr"`
	XSI       string     `xml:"xsi,attr"`
	SchemaLoc string     `xml:"schemaLocation,attr"`
	NS2       string     `xml:"ns2,attr"`
	NS3       string     `xml:"ns3,attr"`
	Metadata  Metadata   `xml:"metadata"`
	Waypoints []Waypoint `xml:"wpt"`
	Tracks    []Track    `xml:"trk"`
}

type Metadata struct {
	Name string `xml:"name"`
	Link Link   `xml:"link"`
	Time string `xml:"time"`
}

type Link struct {
	Href string `xml:"href,attr"`
	Text string `xml:"text"`
}

type Waypoint struct {
	Lat  float64 `xml:"lat,attr"`
	Lon  float64 `xml:"lon,attr"`
	Ele  float64 `xml:"ele"`
	Name string  `xml:"name"`
	Type string  `xml:"type"`
}

type Track struct {
	Name     string     `xml:"name"`
	Segments []TrackSeg `xml:"trkseg"`
}

type TrackSeg struct {
	Points []TrackPoint `xml:"trkpt"`
}

type TrackPoint struct {
	Lat  float64 `xml:"lat,attr"`
	Lon  float64 `xml:"lon,attr"`
	Ele  float64 `xml:"ele"`
	Time string  `xml:"time"`
}

type Point struct {
	Lat float64
	Lon float64
}

const (
	earthRadius = 6378137.0 // Earth's radius in meters
	offsetDist  = 3.0       // Offset distance in meters
)

// findTurnaroundPoint identifies the turnaround point by finding where the track
// starts heading back in the opposite direction
func findTurnaroundPoint(points []TrackPoint) (Point, int) {
	maxDist := 0.0
	var turnaroundIndex int
	var turnaroundPoint Point

	// Calculate distance from start to each point
	startPoint := Point{points[0].Lat, points[0].Lon}

	for i, point := range points {
		dist := math.Pow(point.Lat-startPoint.Lat, 2) + math.Pow(point.Lon-startPoint.Lon, 2)
		if dist > maxDist {
			maxDist = dist
			turnaroundIndex = i
			turnaroundPoint = Point{point.Lat, point.Lon}
		}
	}

	return turnaroundPoint, turnaroundIndex
}

// findClosestTrackSegment finds the two track points that a waypoint lies between
func findClosestTrackSegment(waypoint Point, points []TrackPoint, turnaroundIndex int) (Point, Point) {
	minDist := math.MaxFloat64
	var p1, p2 Point

	// Only look at outbound portion of track
	for i := 0; i < turnaroundIndex; i++ {
		if i+1 >= len(points) {
			break
		}

		// Calculate distance from waypoint to line segment between track points
		curr := Point{points[i].Lat, points[i].Lon}
		next := Point{points[i+1].Lat, points[i+1].Lon}

		dist := distanceToSegment(waypoint, curr, next)
		if dist < minDist {
			minDist = dist
			p1 = curr
			p2 = next
		}
	}

	return p1, p2
}

// distanceToSegment calculates the minimum distance from a point to a line segment
func distanceToSegment(p, start, end Point) float64 {
	// Convert to meters for more accurate distance calculation
	const metersPerDegLat = 111111.0
	px := p.Lon * metersPerDegLat * math.Cos(p.Lat*math.Pi/180.0)
	py := p.Lat * metersPerDegLat
	startx := start.Lon * metersPerDegLat * math.Cos(start.Lat*math.Pi/180.0)
	starty := start.Lat * metersPerDegLat
	endx := end.Lon * metersPerDegLat * math.Cos(end.Lat*math.Pi/180.0)
	endy := end.Lat * metersPerDegLat

	A := px - startx
	B := py - starty
	C := endx - startx
	D := endy - starty

	dot := A*C + B*D
	len_sq := C*C + D*D

	if len_sq == 0 {
		// Start and end points are the same
		return math.Sqrt(A*A + B*B)
	}

	param := dot / len_sq
	var xx, yy float64

	if param < 0 {
		xx = startx
		yy = starty
	} else if param > 1 {
		xx = endx
		yy = endy
	} else {
		xx = startx + param*C
		yy = starty + param*D
	}

	dx := px - xx
	dy := py - yy
	return math.Sqrt(dx*dx + dy*dy)
}

func offsetPoint(p1, p2 Point) Point {
	lat1 := p1.Lat * math.Pi / 180
	lon1 := p1.Lon * math.Pi / 180
	lat2 := p2.Lat * math.Pi / 180
	lon2 := p2.Lon * math.Pi / 180

	dLon := lon2 - lon1
	bearing := math.Atan2(
		math.Sin(dLon)*math.Cos(lat2),
		math.Cos(lat1)*math.Sin(lat2)-math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLon),
	)

	perpBearing := bearing + math.Pi/2

	latOffset := p1.Lat + (offsetDist*math.Cos(perpBearing))/earthRadius*(180/math.Pi)
	lonOffset := p1.Lon + (offsetDist*math.Sin(perpBearing))/(earthRadius*math.Cos(lat1))*(180/math.Pi)

	return Point{
		Lat: latOffset,
		Lon: lonOffset,
	}
}

func createOutboundWaypoints(waypoints []Waypoint, points []TrackPoint, turnaroundPoint Point, turnaroundIndex int) []Waypoint {
	var outboundWaypoints []Waypoint

	for _, wp := range waypoints {
		// Find the track segment this waypoint lies on
		waypointPoint := Point{wp.Lat, wp.Lon}
		trackPoint1, trackPoint2 := findClosestTrackSegment(waypointPoint, points, turnaroundIndex)

		// Calculate offset using the same track segment
		offset := offsetPoint(trackPoint1, trackPoint2)

		// Create the outbound waypoint
		name := wp.Name
		name = strings.ReplaceAll(name, "Rtn", "Out")
		if !strings.Contains(name, "Out") {
			name = strings.TrimSpace(name) + " Out"
		}

		outboundWp := Waypoint{
			Lat:  offset.Lat,
			Lon:  offset.Lon,
			Ele:  wp.Ele,
			Name: name,
			Type: wp.Type,
		}
		outboundWaypoints = append(outboundWaypoints, outboundWp)
	}

	// Add turnaround waypoint
	turnaroundWp := Waypoint{
		Lat:  turnaroundPoint.Lat,
		Lon:  turnaroundPoint.Lon,
		Ele:  0.0,
		Name: "TURN AROUND",
		Type: "GENERAL DISTANCE",
	}
	outboundWaypoints = append(outboundWaypoints, turnaroundWp)

	return outboundWaypoints
}

func processGPX(gpxData *GPX) {
	if len(gpxData.Tracks) == 0 || len(gpxData.Tracks[0].Segments) == 0 {
		fmt.Println("No track data found in GPX file")
		return
	}

	// Get the first track segment
	segment := &gpxData.Tracks[0].Segments[0]

	// Find turnaround point
	turnaroundPoint, turnaroundIndex := findTurnaroundPoint(segment.Points)
	fmt.Printf("Found turnaround point at index %d: %f, %f\n",
		turnaroundIndex, turnaroundPoint.Lat, turnaroundPoint.Lon)

	// Create outbound waypoints
	outboundWaypoints := createOutboundWaypoints(
		gpxData.Waypoints,
		segment.Points,
		turnaroundPoint,
		turnaroundIndex,
	)
	fmt.Printf("Created %d outbound waypoints\n", len(outboundWaypoints))

	// Add the new waypoints to the GPX data
	gpxData.Waypoints = append(gpxData.Waypoints, outboundWaypoints...)

	// Offset outbound track points
	for i := 0; i < turnaroundIndex; i++ {
		if i+1 < len(segment.Points) {
			curr := Point{Lat: segment.Points[i].Lat, Lon: segment.Points[i].Lon}
			next := Point{Lat: segment.Points[i+1].Lat, Lon: segment.Points[i+1].Lon}

			offset := offsetPoint(curr, next)
			segment.Points[i].Lat = offset.Lat
			segment.Points[i].Lon = offset.Lon
		}
	}

	fmt.Printf("Offset %d track points\n", turnaroundIndex)

	fmt.Printf("Finished processing GPX file\n")
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: gpxprocessor <input.gpx> <output.gpx>")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	var gpx GPX
	if err := xml.Unmarshal(data, &gpx); err != nil {
		fmt.Printf("Error parsing GPX: %v\n", err)
		os.Exit(1)
	}

	processGPX(&gpx)

	output, err := xml.MarshalIndent(gpx, "", "  ")
	if err != nil {
		fmt.Printf("Error generating output XML: %v\n", err)
		os.Exit(1)
	}

	output = []byte(xml.Header + string(output))

	if err := os.WriteFile(outputFile, output, 0644); err != nil {
		fmt.Printf("Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully wrote modified GPX to %s\n", outputFile)
}
