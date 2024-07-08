# virtualclock
virtual clock function for federation

## ARIA project
ARIA is a federation platform that simultaneously operates disaster-related systems and simulations.

## Badge
<img src="https://github.com/ARIA-x/evacuation-simulator/assets/12294073/9ecfefd8-1187-4ae7-bdd5-f182a3a34ca4" width="7%">
This works as expected in a specific setting

## Usage
docker-compose.yaml at the top directory can run every test code in each container.
or you can run each test codes with docker-compose.yaml in each directory

Basically the following three steps is needed to run the virtual clock service.
- Start MQTT broker.
- Start the vcserver.
- Run the simulator with the VClock library

##Sample
Sample code includes a simple differential equation simulation.
