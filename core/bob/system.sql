-- GetTimeZone *TimeZoneDB:TimeZoneDBSlice:bob.SliceTransformer[*TimeZoneDB, TimeZoneDBSlice]
SELECT name, abbrev FROM pg_timezone_names WHERE name = current_setting('TIMEZONE');
