# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

-

## [1.3] - 2023-05-04

### Fixed

- Show Warning if GPX file does not have time stamps
- Fix bad condition to get closes GPS
- Fix elevation when has decimal values
- Fix GPX file has time stamps out of order
- Fix GPX timezone

### Added

- Show a notice when gps data is updated
- Add an option to show more verbose information
- Add default country support to remove geolocation
- Update tracks with geolocation, date and time
- Add support for store the old filename in header.name
- Check if the gpx file has positions further away than 1km
- Adds support to search images with GPS positioning too
- Warns if the distance is greater than 500 meters in less than 30 seconds

## [1.2] - 2023-05-01

### Fixed

- Restructures the code in directories
- Fix use of global variable
- Fix problem when exif data is not obtained if GPS time is obtained from mp4

### Added

- Include support for read GPS Time from Gopro Video
- Support for read any video or image file
- show date comparison
- Get GPS Time from image file
- Show kind of time (Archive time, Exit time, GPS time)

## [1.1] - 2023-04-20

### Fixed

- Fix load GPX files
- Fix store elevation
- Update dependencies
- Fix small errors

### Added

- Support for geocoding using openstreetmap
- New parameter to force update even overwriting previous GPS data
- Support for multiples GPX files
- Use color red for fatal errors
- Show old location
- Support to adjust the date according to the time zone and if it has daylight saving time

## [1.0] - 2023-04-15

- Initial version

[Unreleased]: https://github.com/inode64/SyncMediaTrack/compare/v1.1...main
[1.1]: https://github.com/inode64/SyncMediaTrack/compare/v1.0...v1.1
[1.2]: https://github.com/inode64/SyncMediaTrack/compare/v1.1...v1.2
[1.3]: https://github.com/inode64/SyncMediaTrack/compare/v1.2...v1.3
