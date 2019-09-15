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

**or the miner will cannot connect to the server and the sol will not be relayed to the node!**

### TODO
- Web UI
- more accurate hashrate
- more
