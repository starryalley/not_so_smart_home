My own little not so smart home automation project

# Purpose

I just installed the following gadgets on my raspi over the weekend:
- SSD1306 display
- TSL2561 light sensor
- DHT22 temperature and humidity sensor
- a RGB LED (common anode type)

Also I have a Xiaomi Smart Home gateway with the following connected sensors
- 2 door sensors (front and back)
- 1 human body sensor placed in the kitchen
- 1 smart plug to control a floor lamp in the living room

I just want to have something else in addition to what Xiaomi's app can do.

# Current automation

## Temperature to RGB Light

Based on current temperature in the room, the color of the RGB LED will change accordingly, where blue means cold, and red means warm. 


## Turn on floor lamp automatically

If it's already sunset and no one is home (dark in the living room), the floor lamp will turn on automatically until midnight.

When living room is bright (someone is there) the floor lamp will turn off as well.


## Save sensors data to Google spreadsheet

The script will also upload temperature/humidity/luminosity data into a private google sheet for record.


# TODO

I still can't figure out if there is anything else I can do with the sensors I got. Guess it's all for now.

