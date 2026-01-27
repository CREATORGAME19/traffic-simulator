const { select, zoom, zoomIdentity } = d3;
const SVG_NS = "http://www.w3.org/2000/svg";

const SIM_RATES = [0,0.5,1,2,3,4,5,10,100];

const closePopup = () => {
    document.getElementById('map').removeEventListener('click', closePopup);
    const popup = document.getElementById("active-popup");
    if (popup != null) {
        popup.remove();
    }
};
class Renderer {
    constructor(web_socket) {
        this.vehicle_data = [];
        this.node_data = [];
        this.lane_data = [];
        this.curr_time = 0;
        this.prev_sim_rate = 1;
        this.sim_rate = 0;
        this.web_socket = web_socket;

        this.scaleFactor = 0.25;
        this.xScaleOffset = 0;
        this.yScaleOffset = 0;

        this.currentTransform = zoomIdentity;

        this.initLayers();

        this.zoomBehavior = zoom()
            .scaleExtent([0.5, 10])
            .filter(event => !event.target.closest("#controls-layer"))
            .on('zoom', this.handleZoom.bind(this));

        this.initZoom();
        this.center();
        const div = document.querySelector("#map");

        const observer = new ResizeObserver(entries => {
        for (const entry of entries) {
            this.onResize();
        }
        });

        observer.observe(div);
    }

    initLayers() {
        const svg = select('svg');

        svg.append('g')
            .attr('id', 'map-layer');

        svg.append('g')
            .attr('id', 'vehicle-layer');

        svg.append('g')
            .attr('id', 'controls-layer');
    }

    handleZoom(e) {
        select('#map-layer')
        .attr('transform', e.transform);
        select('#vehicle-layer')
        .attr('transform', e.transform);
        this.currentTransform = e.transform;
        this.renderVehicleUpdate();
        this.updateSvgPopup();
    }

    initZoom() {
        select('svg')
        .call(this.zoomBehavior);
    }

    center() {
        select('svg')
        .transition()
        .call(this.zoomBehavior.translateTo, 0, 0);
    }

    updateNodes(n) {
        this.node_data = n;
    }

    updateLanes(l) {
        this.lane_data = l;
    }

    updateVehicles(v) {
        this.vehicle_data = v;
    }

    updateCurrTime(t) {
        this.curr_time = t;
    }

    updateSimRate(s) {
        this.sim_rate = s;
    }

    formatTime(seconds) { //Input in seconds
        const hrs = Math.floor(seconds / 3600);
        const mins = Math.floor((seconds % 3600) / 60);
        const secs = Math.floor(seconds % 60);

        const pad = n => String(n).padStart(2, '0');

        return `${pad(hrs)}:${pad(mins)}:${pad(secs)}`;
    }

    renderControls() {
        const map = document.getElementById("map");
        const rect = map.getBoundingClientRect();

        var foreignObjectContainer = document.getElementById("topLeftOverlay");
        if (foreignObjectContainer == null) {
            var foreignObjectContainer = document.createElementNS(SVG_NS, "foreignObject");
            foreignObjectContainer.setAttribute("id", "topLeftOverlay");
            foreignObjectContainer.setAttribute("width", 135); //No dynamic resizing :(
            foreignObjectContainer.setAttribute("height",26); //No dynamic resizing :(
            foreignObjectContainer.setAttribute("x", 10);
            foreignObjectContainer.setAttribute("y", 20);
            
            foreignObjectContainer.innerHTML = `
            <div id="topLeftPanel" xmlns="http://www.w3.org/1999/xhtml">
                <div id="timeIndicator" width="fit-content">Time: </div>
            </div>`;

            document.getElementById("controls-layer").appendChild(foreignObjectContainer);
        }
        document.getElementById("timeIndicator").innerHTML = 'Time: '+this.formatTime(this.curr_time);

        this.renderBottomControlsOverlay(rect);
    }

    renderBottomControlsOverlay(rect) {
        var foreignObjectContainer = document.getElementById("bottomControlsOverlay");
        if (foreignObjectContainer == null) {
            var foreignObjectContainer = document.createElementNS(SVG_NS, "foreignObject");
            foreignObjectContainer.setAttribute("id", "bottomControlsOverlay");
            foreignObjectContainer.setAttribute("width", 150);
            foreignObjectContainer.setAttribute("height",70);
            
            foreignObjectContainer.innerHTML = `
            <div id="bottomControlsPanel" xmlns="http://www.w3.org/1999/xhtml">
                <button id="pauseBtn">⏸</button>
                <button id="simRateDecBtn">-</button>
                <p id="simRateIndicator"></p>
                <button id="simRateIncBtn">+</button>
            </div>`;

            document.getElementById("controls-layer").appendChild(foreignObjectContainer);
            document.getElementById("simRateIncBtn").addEventListener("click", () => { this.SimRateIncrease() });
            document.getElementById("simRateDecBtn").addEventListener("click", () => { this.SimRateDecrease() });
            document.getElementById("pauseBtn").addEventListener("click", () => { this.PauseSimRate() });
        }
        foreignObjectContainer.setAttribute("x", (rect.width*0.5) - 70);
        foreignObjectContainer.setAttribute("y", rect.height-70);
        document.getElementById("simRateIndicator").innerHTML = this.sim_rate+'x';
    }

    renderStaticMap() {
        for (let i=0;i<this.lane_data.length;i++) {
            var lane = document.getElementById("lane_"+String(i));
            if (lane == null) {
                var lane = document.createElementNS(SVG_NS, "line");
                lane.setAttribute("id", "lane_"+String(i));
                lane.setAttribute("x1", this.getRenderX(this.lane_data[i].Start_X));
                lane.setAttribute("y1", this.getRenderY(this.lane_data[i].Start_Y));
                lane.setAttribute("x2", this.getRenderX(this.lane_data[i].End_X));
                lane.setAttribute("y2", this.getRenderY(this.lane_data[i].End_Y));

                document.getElementById("map-layer").appendChild(lane);
            }
        }
        for (let i=0;i<this.node_data.length;i++) {
            var node = document.getElementById("node_"+String(i));
            if (node == null) {
                var node = document.createElementNS(SVG_NS, "circle");
                node.setAttribute("id", "node_"+String(i));
                node.setAttribute("cx", this.getRenderX(this.node_data[i].X));
                node.setAttribute("cy", this.getRenderY(this.node_data[i].Y));
                node.setAttribute("style", 'pointer-events: all;cursor: pointer;');
                node.addEventListener("click", () => { 
                    this.createSvgPopup("node", i, `Node ${i}`, `X: ${this.node_data[i].X || 0}, Y: ${this.node_data[i].Y || 0}`);
                });
                switch (this.node_data[i].AgentType) {
                    case 0:
                        var class_name = 'intersection-node';
                        var radius = 16;
                        break;
                    case 1:
                        var class_name = 'spawner-node';
                        var radius = 5;
                        break;
                    case 2:
                        var class_name = 'sink-node';
                        var radius = 5;
                        break;
                    case 3:
                        var class_name = 'traffic-light-node';
                        var radius = 16;
                        break;
                    default:
                        var class_name = 'road-node';
                        var radius = 5;
                }
                node.setAttribute("class", class_name);
                node.setAttribute("r",radius);

                document.getElementById("map-layer").appendChild(node);
            }
        }
    }

    renderVehicleUpdate() {
        const t = this.currentTransform;
        for (let i=0;i<this.vehicle_data.length;i++) {
            var vehicle = document.getElementById("vehicle_"+String(i));
            if (vehicle == null) {
                var vehicle = document.createElementNS(SVG_NS, "image");
                vehicle.setAttribute("id", "vehicle_"+String(i));
                vehicle.setAttribute("href", '/static/images/bus-logo.png');
                vehicle.setAttribute("style", 'pointer-events: all;cursor: pointer;');
                vehicle.addEventListener("click", () => { 
                    this.createSvgPopup("vehicle", i, `Vehicle ${i}`, `X: ${this.vehicle_data[i].X || 0}, Y: ${this.vehicle_data[i].Y || 0}`);
                });

                document.getElementById("vehicle-layer").appendChild(vehicle);
            }
            vehicle.setAttribute("x", this.getRenderX(this.vehicle_data[i].X)-(15/t.k));
            vehicle.setAttribute("y", this.getRenderY(this.vehicle_data[i].Y)-(15/t.k));
            vehicle.setAttribute("width", 30/t.k);
            vehicle.setAttribute("height", 30/t.k);
            vehicle.classList.toggle("invisible-vehicle", this.vehicle_data[i].Status == 0);
        }
        this.updateSvgPopup();
    }

    getRenderX(world_x) {
    return world_x*this.scaleFactor + this.xScaleOffset;
    }

    getRenderY(world_y) {
    return world_y*-1*this.scaleFactor + this.yScaleOffset;
    }

    onResize() {
        this.renderControls();
    }

    SimRateIncrease() {
        var nearest_index = 0;
        while (SIM_RATES[nearest_index] <= this.sim_rate) {
            nearest_index++;
        }
        if (nearest_index >= SIM_RATES.length) {
            nearest_index = 0;
        }
        this.updateSimRate(SIM_RATES[nearest_index]);
        this.renderControls();
        this.sendSimRateUpdateMsg(SIM_RATES[nearest_index]);
        if (this.sim_rate == 0) {
            document.getElementById("pauseBtn").innerHTML = "▶";
        } else {
            document.getElementById("pauseBtn").innerHTML = "⏸";
        }
    }

    SimRateDecrease() {
        var nearest_index = SIM_RATES.length-1;
        while (SIM_RATES[nearest_index] >= this.sim_rate) {
            nearest_index--;
        }
        if (nearest_index < 0) {
            nearest_index = SIM_RATES.length-1;
        }
        this.updateSimRate(SIM_RATES[nearest_index]);
        this.renderControls();
        this.sendSimRateUpdateMsg(SIM_RATES[nearest_index]);
        if (this.sim_rate == 0) {
            document.getElementById("pauseBtn").innerHTML = "▶";
        } else {
            document.getElementById("pauseBtn").innerHTML = "⏸";
        }
    }

    PauseSimRate() {
        if (this.sim_rate != 0) {
            this.prev_sim_rate = this.sim_rate != 0 ? this.sim_rate : 1;
            this.updateSimRate(0); 
            this.sendSimRateUpdateMsg(0);
            
        } else {
            this.updateSimRate(this.prev_sim_rate); 
            this.sendSimRateUpdateMsg(this.prev_sim_rate);
        }
        if (this.sim_rate == 0) {
            document.getElementById("pauseBtn").innerHTML = "▶";
        } else {
            document.getElementById("pauseBtn").innerHTML = "⏸";
        }
        this.renderControls();
    }

    sendSimRateUpdateMsg(new_sim_rate) {
        this.web_socket.send(
            JSON.stringify(
                {Type: "sim_rate_update", Data: new_sim_rate}
            )
        );
    }

    updateSvgPopup() {
        const popup = document.getElementById("active-popup");
        if (popup == null) return;

        const t = this.currentTransform;
        var x = 0.0;
        var y = 0.0;
        switch (popup.dataset.type){
            case "node":
                x = this.getRenderX(this.node_data[parseInt(popup.dataset.id)].X);
                y = this.getRenderY(this.node_data[parseInt(popup.dataset.id)].Y);
                
                break;
            case "vehicle":
                x = this.getRenderX(this.vehicle_data[parseInt(popup.dataset.id)].X);
                y = this.getRenderY(this.vehicle_data[parseInt(popup.dataset.id)].Y);
                break;
            default:
                break;
        }

        // Inverse scale factor
        const s = 1 / t.k;

        const xPos = x - (65 * s);
        const yPos = y - (70 * s);

        popup.setAttribute("transform", `translate(${xPos}, ${yPos}) scale(${s})`);
    }

    createSvgPopup(item_type, item_id, title, content) {
        const parentSvg = document.getElementById("vehicle-layer");
        
        var oldPopup = document.getElementById("active-popup");
        if (oldPopup != null) {
            closePopup();
        }
        
        const group = document.createElementNS(SVG_NS, "g");
        group.setAttribute("id", "active-popup");
        
        const bgColor = "rgba(255, 255, 255, 0.9)";

        group.dataset.type = item_type;
        group.dataset.id = item_id;

        const rect = document.createElementNS(SVG_NS, "rect");
        rect.setAttribute("width", "130");
        rect.setAttribute("height", "55");
        rect.setAttribute("rx", "6");
        rect.setAttribute("fill", bgColor);
        rect.setAttribute("stroke", "rgba(0, 0, 0, 0.1)");
        rect.setAttribute("stroke-width", "1");

        const text = document.createElementNS(SVG_NS, "text");
        text.setAttribute("x", "15");
        text.setAttribute("y", "22");
        text.setAttribute("fill", "#000");
        text.setAttribute("style", "font-family: sans-serif; font-size: 13px; font-weight: bold;");
        text.textContent = title;

        const subText = document.createElementNS(SVG_NS, "text");
        subText.setAttribute("x", "15");
        subText.setAttribute("y", "42");
        subText.setAttribute("fill", "#333");
        subText.setAttribute("style", "font-family: sans-serif; font-size: 11px;");
        subText.textContent = content;

        group.appendChild(rect);
        group.appendChild(text);
        group.appendChild(subText);
        parentSvg.appendChild(group);

        this.updateSvgPopup();

        setTimeout(() => {document.getElementById("map").addEventListener('click', closePopup)}, 10);
    }
}