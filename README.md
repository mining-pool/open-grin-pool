# open-grin-pool

this pool is originally designed for epic (=[epicash](http://epic.tech)). And the codebase of epic is grin so it can be generally used as the grin pool. 

### features
- relay the miner conn to the grin node, totally native experience
- expose the TUI miner detail to http api
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

Maintainer can manually use this command to send the coin `grin wallet send -d http://<IP>:<PORT>`. Note, ensure the receiver online before your sending.

### Config

#### For server

```toml
[server.stratum_mining_config]

#whether stratum server is enabled
enable_stratum_server = true

#what port and address for the stratum server to listen on
stratum_server_addr = "127.0.0.1:3416"

#the amount of time, in seconds, to attempt to mine on a particular
#header before stopping and re-collecting transactions from the pool
attempt_time_per_block = 15 # Should be shorter
cuckatoo_minimum_share_difficulty = 3 # Should be a little bit larger
randomx_minimum_share_difficulty = 5000 # Should be a little bit larger
progpow_minimum_share_difficulty = 100000 # Should be a little bit larger

#the wallet receiver to which coinbase rewards will be sent
wallet_listener_url = "http://127.0.0.1:3415"

#whether to ignore the reward (mostly for testing)
burn_reward = false

```

if you are using epic you can keep all default except `auth_pass`. The password can be found in the `.api_secret` file. 
    
knowledge about this, check [here](https://github.com/mimblewimble/grin/blob/master/doc/api/api.md)

#### For miner

In miner's config(a `.toml` config file), these 2 params are **required**

```toml
# login for the stratum server (if required)
stratum_server_login = "loginName"

# password for the stratum server (if required)
stratum_server_password = "loginPass"
```

**or the miner will cannot connect to the server and the sol will not be relayed to the node!**

### TODO
- Web UI
- more accurate hashrate
- more
