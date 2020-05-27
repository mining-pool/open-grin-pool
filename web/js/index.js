function gotoMinerPage() {
    login = document.getElementById("login").value;
    // if (login===null){return}
    window.location.href = "/miner.html?login=" + login;
}

document.getElementById("button").addEventListener("click", gotoMinerPage);

function get_pool_status() {
    let xhr = new XMLHttpRequest();
    xhr.open('GET', API + "/pool", true);
    xhr.responseType = 'json';
    xhr.onload = function () {
        let status = xhr.status;
        if (status === 200) {
            let pool = xhr.response;
       
            document.getElementById("conn").innerText = pool.node_status.connections;
            document.getElementById("height").innerText = pool.node_status.tip.height;
            document.getElementById("count").innerText = pool.mined_blocks.length;
            document.getElementById("cuckatoo").innerText = pool.node_status.tip.total_difficulty.cuckatoo;
            document.getElementById("progpow").innerText = pool.node_status.tip.total_difficulty.progpow;
            document.getElementById("randomx").innerText = pool.node_status.tip.total_difficulty.randomx;
        } else {
            console.log(xhr.response)
        }
    };

    xhr.send();
}


function get_blocks() {
    let xhr = new XMLHttpRequest();
    xhr.open('GET', API + "/blocks", true);
    xhr.responseType = 'json';
    xhr.onload = function () {
        let status = xhr.status;
        if (status === 200) {
            let blocks = xhr.response;
    
            let divBlocks = document.getElementById("blocks");
            for (i = 0; i < blocks.length; i++) {
                let node = document.createElement("li");
                node.className = "list-group-item";
                let textnode = document.createTextNode(blocks[i]);
                node.appendChild(textnode);
                divBlocks.appendChild(node)
            }

        } else {
            console.log(xhr.response)
        }
    };

    xhr.send();
}


get_pool_status();
get_blocks();