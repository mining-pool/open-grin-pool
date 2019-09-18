document.getElementById("login").innerText = window.location.search.substr(7);

let xhr = new XMLHttpRequest();
xhr.open('GET', API + "/miner/" + window.location.search.substr(7), true);
xhr.responseType = 'json';
xhr.onload = function () {
    let status = xhr.status;
    if (status === 200) {
        let miner = xhr.response;

        document.getElementById("totalAHS").innerText = (miner.average_hashrate / 1000) + "kh/s";
        document.getElementById("totalRHS").innerText = (miner.realtime_hashrate / 1000) + "kh/s";

        agents = document.getElementById("agents");

        for (const agent in miner.agents) {
            let name_node = document.createElement("li");
            let name_textnode = document.createTextNode(agent);
            name_node.appendChild(name_textnode);
            agents.appendChild(name_node);

            let node = document.createElement("li");
            let textnode = document.createTextNode(JSON.stringify(miner.agents[agent]));
            node.appendChild(textnode);

            agents.appendChild(node)
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