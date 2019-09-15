document.getElementById("login").innerText = window.location.search.substr(7);

let xhr = new XMLHttpRequest();
xhr.open('GET', API + "/miner" + window.location.search.substr(7), true);
xhr.responseType = 'json';
xhr.onload = function () {
    let status = xhr.status;
    if (status === 200) {
        let miner = JSON.parse(xhr.responseText);

        document.getElementById("ths").innerText = miner.hashrate;
        agents = document.getElementById("agents");

        miner.agents.forEach(agent => {
            let node = document.createElement("li");
            let textnode = document.createTextNode(JSON.stringify(agent));
            node.appendChild(textnode);
            agents.appendChild(node)
        });
    } else {
        console.log(xhr.response)
    }
};

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
            alert(xhr.responseText)
        } else {
            console.log(xhr.response)
        }
    };
    xhr.send(json);
}