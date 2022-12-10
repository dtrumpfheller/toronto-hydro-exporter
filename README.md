# Toronto Hydro Exporter
Exports hourly electric meter data from Toronto Hydro to an InfluxDB.

## Configuration
The configuration file must be a valid YAML file. Its path can be passed into the application as an argument, else **config.yml** is assumed.

Example **config.yml** file:
```
influxDB:
  url: http://192.168.0.252:9086
  token: abc
  organization: home
  bucket: torontohydro
torontoHydro:
  username: <username>
  password: <password>
sleepDuration: 720
lookDaysInPast: 1
```

| Name                     | Description                                                                 |
|--------------------------|-----------------------------------------------------------------------------|
| influxDB.url             | address of InfluxDB2 server                                                 |
| influxDB.token           | auth token to access InfluxDB2 server                                       |
| influxDB.organization    | organization of InfluxDB2 server                                            |
| influxDB.bucket          | name of bucket                                                              |
| torontoHydro.username    | used to log into Toronto Hydro                                              |
| torontoHydro.password    | used to log into Toronto Hydro                                              |
| sleepDuration            | sleep time between exports in minutes, zero means run only once             |
| lookDaysInPast           | how many days of the past should be considered                              |

## Docker
The exporter was written with the intent of running it in docker. You can also run it directly if this is preferred.

### Build Image
Execute following statement, then either start via docker or docker compose.
```
docker build -t toronto-hydro-exporter .
```

### Docker
```
docker run -d --restart unless-stopped --name=toronto-hydro-exporter -v ./config.yml:/config.yml toronto-hydro-exporter
```

### Docker Compose
```
version: "3.4"
services:
  toronto-hydro-exporter:
    image: toronto-hydro-exporter
    container_name: toronto-hydro-exporter
    restart: unless-stopped
    volumes:
      - ./config.yml:/config.yml:ro
```