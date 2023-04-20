# SyncMediaTrack
Synchronize Media Data from track GPX

Getting started with SyncMediaTrack

## 1) Set-up your Camera

Set the local time and date of your camera precisely before shooting.

Alternatively you can set your camera time to GMT (http://wwp.greenwichmeantime.com/) which can be handy as you won’t have to set summer/winter time or when you travel through time zones.

## 2) Take pictures while recording a tracklog

Make sure your GPS receiver is recording a track log. Keep your GPSr ON during all the time you take pictures.

## 3) Download exiftool 

Get the latest version from https://exiftool.org/ or install it from the installer of your Linux distribution.
In **Windows** you need to copy the _exiftool_ executable to some directory included in the %PATH% environment variable, for example c:\Windows.

## 4) Run from command-line SyncMediaTrack
First it is advisable to check that the images are well located,
```
SyncMediaTrack --dry-run --geoservice --track XXXX.gpx photos/Andorra
```
Once we have verified that everything is correct we can execute it again adding the correct geographic positions
```
SyncMediaTrack --track XXXX.gpx photos/Andorra
```

---

# Trouble Shooting

#### No indications of localization appears

Check your gpx file first then your pictures this way.

You can open and modify your GPX file easily in a text editor: it has plenty of tags
```
<tag>
....
</tag>
```

Gpx files have one or more

```
<trkseg>
...
</trkseg>
```
which contain plenty of trackpoints
```
<trkpt>
...
</trkpt>
```
which says at which location you were at a precise time.
```
...
<trkseg>
  ... 
  <trkpt lat="48.50517319" lon="7.13916969">
    <ele>700.46</ele>
    <time>2007-03-04T12:05:16Z</time>
  </trkpt>
  <trkpt lat="48.50517286" lon="7.13916759">
    <ele>704.30</ele>
    <time>2007-03-04T12:05:18Z</time>
  </trkpt>
  <trkpt lat="48.50517294" lon="7.13916550">
    <ele>708.15</ele>
    <time>2007-03-04T12:05:20Z</time>
  </trkpt
  ... 
</trkseg>
...
```