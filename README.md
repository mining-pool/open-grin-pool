# open-grin-pool
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fmaoxs2%2Fopen-grin-pool.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fmaoxs2%2Fopen-grin-pool?ref=badge_shield) [![Build Status](https://travis-ci.org/maoxs2/open-grin-pool.svg?branch=master)](https://travis-ci.org/maoxs2/open-grin-pool) [![Go Report Card](https://goreportcard.com/badge/github.com/maoxs2/open-grin-pool)](https://goreportcard.com/report/github.com/maoxs2/open-grin-pool)

This is originally designed for the epic (=[epicash](http://epic.tech)). And the codebase of the epic is grin so it can be generally used as the grin pool.

### features
- relay the miner conn to the grin node, totally native experience
- expose the TUI miner detail to Http API
- backup the share(submit) histories per specific time interval
- record the miner's pay method (manually sent coin by pool maintainer)
- pool status

### Usage

```bash
# if grin
grin server run 
# if epic
epic server run

# ready
git clone https://github.com/maoxs2/open-grin-pool.git pool
cd pool && go build .

# config
vi config.json

# start
./open-grin-pool 

```

WebAPI:
- `/pool` basic pool status
- `/revenue` the revenue **last day**, which the pool maintainer has to sent **today**
- `/shares` the all miners' shares **today**
- `/miner/{miner_login}` GET is the miner status
POST upload the payment method. e.g. ` curl 127.0.0.1:3333/miner/Hello` will get the json of "Hello"'s status. `curl  -X POST -d "{'pass': 'passwordOfHello', 'pm': 'http://<IP>:<PORT>'}" 127.0.0.1:3333/miner/Hello`

The maintainer can manually use this command to send the coin `grin wallet send -d http://<IP>:<PORT>`. Note, ensure the receiver online before your sending.

WebPage:
- replace the API address to your API in the web/config.js
- move all files in web to your Nginx/Apache folder

### Config

#### For server

If you are using epic you can keep all default except `auth_pass`. The password can be found in the `.api_secret` file. 
    
knowledge about this, check [here](https://github.com/mimblewimble/grin/blob/master/doc/api/api.md)

The `diff` should be the same difficulty to what you configured in server's (not pool's) `.toml` config file.


#### For miner

In miner's config(a `.toml` config file), these 2 params are **required**

```toml
# login for the stratum server (if required)
stratum_server_login = "loginName"

# password for the stratum server (if required)
stratum_server_password = "loginPass"
```

The `agent` is optional. Pool treat it as the miner's rig name.

**or the miner will not be able to connect to the server and the sol will not be relayed to the node!**

### TODO
- Web UI
- More accurate hash rate
- Administration based on web
- multi Backend (failover)
- multi Port (with different difficulties)

Please **donation**:
**LTC: LXxqHY4StG79nqRurdNNt1wF2yCf4Mc986**

## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fmaoxs2%2Fopen-grin-pool.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fmaoxs2%2Fopen-grin-pool?ref=badge_large)
