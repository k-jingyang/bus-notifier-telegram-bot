#!/bin/bash
LTA_API_TOKEN=$(grep LTA_API_TOKEN ../.env | cut -d '=' -f 2-)

curl -H "AccountKey: $LTA_API_TOKEN" http://datamall2.mytransport.sg/ltaodataservice/BusServices \
| jq -r .value[].ServiceNo \
| uniq > ../bus_services.txt


go run data_downloader.go
