# lastfmmastodon
Toot your weekly and/or yearly last.fm stats to Mastodon

This app is written with the assumption that you're Tooting from your own account rather than a bot account (although the steps might be similar).

- First time run lastfmmastodon -r - this will save off your access token
- For last.fm get your key and secret at: https://www.last.fm/api/account/create (more about their API at: https://www.last.fm/api)
- At $HOME/.config/lastfmmastodon you should have a secrets.json file that looks like:


```json

{
        "lastfm":
                {
                        "key": "last.fm key",
                        "secret": "last.fm secret",
                        "username": "last.fm username"
                },
        "mastodon":
            {
                    "access_token": "Mastodon Access Token",
                    "api_base_url": "URL of your Mastodon instance"
            }
}


```

Everything should be ready to go.