# open-grin-pool

this pool is originally designed for epic (=[epicash](http://epic.tech)). And the codebase of epic is grin so it can be generally used as the grin pool. 

### features
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
- `/pool` pool status
- `/revenue` the revenue today
- `/miner/{miner_login}` GET is the miner status, POST upload the payment method

### TODO
- Web UI
- payment
- more accurate hashrate
- more
