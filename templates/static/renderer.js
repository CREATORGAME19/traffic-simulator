const { select, zoom, zoomIdentity } = d3;
const SVG_NS = "http://www.w3.org/2000/svg";

const SIM_RATES = [0,0.5,1,2,3,4,5,10,100];

class Renderer {
    constructor(web_socket) {
        this.vehicle_data = [];
        this.node_data = [];
        this.lane_data = [];
        this.curr_time = 0;
        this.prev_sim_rate = 0;
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

        var time_display = document.getElementById("time_indicator");
        if (time_display == null) {
            var time_display = document.createElementNS(SVG_NS, "text");
            time_display.setAttribute("x", 10);
            time_display.setAttribute("y", 20);
            time_display.setAttribute("font-size", 21);
            time_display.setAttribute("id", "time_indicator");

            document.getElementById("controls-layer").appendChild(time_display);
        }
        time_display.textContent = 'Time: '+this.formatTime(this.curr_time);

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

                document.getElementById("vehicle-layer").appendChild(vehicle);
            }
            vehicle.setAttribute("x", this.getRenderX(this.vehicle_data[i].X)-(15/t.k));
            vehicle.setAttribute("y", this.getRenderY(this.vehicle_data[i].Y)-(15/t.k));
            vehicle.setAttribute("width", 30/t.k);
            vehicle.setAttribute("height", 30/t.k);
            vehicle.setAttribute("opacity", this.vehicle_data[i].Status);
        }
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
        if (this.sim_rate == 0) {
            document.getElementById("pauseBtn").innerHTML = "⏸";
        }
        this.updateSimRate(SIM_RATES[nearest_index]);
        this.renderControls();
        this.sendSimRateUpdateMsg(SIM_RATES[nearest_index]);
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
}