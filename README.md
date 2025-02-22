# Silly Bot

Born out of a random idea on Discord, to make a channel only available on odd hours.

## Installation

The hosted version uses a 1 hour interval, so every other hour people won't be able to see the channel(s). State is checked per channel, so you could even create a setup where `#circle` is visible at even hours and `#square` is visible at odd hours.

1. Invite the bot to your server: https://discord.com/oauth2/authorize?client_id=1342582112666255420
2. For the channel(s) that you want to enable, add an advanced permission overwrite for the bot. Enable 'View Channel', 'Manage Channel' and 'Manage Permissions'
3. Watch the chaos unfold

## Run it yourself

Either copy .env.dist to .env and fill in the required variables, or make sure they are available as environment variables.


If you have go installed, just run it like this:

```
go run .
```

Or run it in Docker

## Todo

- [x] Fix drift
- [x] Add docker file
- [ ] Add handler to connect new servers without needing a restart
- [ ] Fix drift for real
