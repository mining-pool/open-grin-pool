document.getElementById("login").innerText = window.location.search.substr(7);

let xhr = new XMLHttpRequest();
xhr.open('GET', API + "/miner/" + window.location.search.substr(7), true);
xhr.responseType = 'json';
xhr.onload = function () {
    let status = xhr.status;
    if (status === 200) {
        let miner = xhr.response;

        agents = document.getElementById("agents");

        let average_hashrate = 0, realtime_hashrate = 0;

        for (const agent in miner.agents) {
            let name_node = document.createElement("li");
            let name_textnode = document.createTextNode(agent);
            name_node.appendChild(name_textnode);
            agents.appendChild(name_node);

            let node = document.createElement("li");
            let textnode = document.createTextNode(JSON.stringify(miner.agents[agent]));
            average_hashrate = average_hashrate + miner.agents[agent].average_hashrate;
            realtime_hashrate = realtime_hashrate + miner.agents[agent].realtime_hashrate;
            node.appendChild(textnode);

            agents.appendChild(node);

            console.log(miner.agents[agent].average_hashrate);
            console.log(miner.agents[agent].realtime_hashrate);
            document.getElementById("totalAHS").innerText = (average_hashrate / 1000) + " kh/s";
            document.getElementById("totalRHS").innerText = (realtime_hashrate / 1000) + " kh/s";
            document.getElementById("lastshare").innerText = miner.lastShare;
        }
    } else {
        console.log(xhr.response)
    }
};

xhr.send();

function registerPayment() {
    let pass = document.getElementById("pass").value;
    let pm = document.getElementById("pm").value;
    let json = JSON.stringify({pass: pass, pm: pm});
    let xhr = new XMLHttpRequest();
    xhr.open('POST', API + "/miner" + window.location.search.substr(7), true);
    xhr.responseType = 'json';
    xhr.onload = function () {
        let status = xhr.status;
        if (status === 200) {
            alert(xhr.response)
        } else {
            console.log(xhr.response)
        }
    };
    xhr.send(json);
}
