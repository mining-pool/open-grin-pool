function gotoMinerPage() {
    login = document.getElementById("login").value;
    // if (login===null){return}
    window.location.href = "/miner.html?login=" + login
}

document.getElementById("button").addEventListener("click", gotoMinerPage);

let xhr = new XMLHttpRequest();
xhr.open('GET', API + "/pool", true);
xhr.responseType = 'json';
xhr.onload = function () {
    let status = xhr.status;
    if (status === 200) {
        let pool = JSON.parse(xhr.responseText);

        let divBlocks = document.getElementById("blocks");
        for (i = 0; i < pool.mined_blocks.length; i++) {
            let node = document.createElement("li");
            let textnode = document.createTextNode(pool.mined_blocks[i]);
            node.appendChild(textnode);
            divBlocks.appendChild(node)
        }

        document.getElementById("conn").innerText = pool.node_status.connections;
        document.getElementById("height").innerText = pool.node_status.tip.height;
        document.getElementById("total_difficulty").innerText = pool.node_status.tip.total_difficulty
    } else {
        console.log(xhr.response)
    }
};

xhr.send();
