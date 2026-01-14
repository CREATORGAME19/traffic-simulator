const { select, zoom, zoomIdentity } = d3;

class Renderer {
    constructor() {
        this.vehicle_data = [];
        this.node_data = [];
        this.lane_data = [];

        this.scaleFactor = 0.25;
        this.xScaleOffset = 0;
        this.yScaleOffset = 0;
        let width = document.querySelector('.map').offsetWidth;
        let height = document.querySelector('.map').offsetHeight;

        this.currentTransform = zoomIdentity;

        this.initLayers();

        this.zoomBehavior = zoom()
            .scaleExtent([0.5, 10])
            .on('zoom', this.handleZoom.bind(this));

        this.initZoom();
        this.center();
    }

    initLayers() {
        const svg = select('svg');

        svg.append('g')
            .attr('id', 'map-layer');

        svg.append('g')
            .attr('id', 'vehicle-layer');
    }

    handleZoom(e) {
        select('#map-layer')
        .attr('transform', e.transform);
        select('#vehicle-layer')
        .attr('transform', e.transform);
        this.currentTransform = e.transform;
        this.vehicleZoomUpdate();
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

    renderStaticMap() {
        select('#map-layer')
        .selectAll('line')
        .data(this.lane_data, d => d.Lane_ID)
        .join('line')
        .attr('x1', d => this.getRenderX(d.Start_X))
        .attr('y1', d => this.getRenderY(d.Start_Y))
        .attr('x2', d => this.getRenderX(d.End_X))
        .attr('y2', d => this.getRenderY(d.End_Y));
        select('#map-layer')
        .selectAll('circle')
        .data(this.node_data, d => d.Node_ID)
        .join('circle')
        .attr('cx', d => this.getRenderX(d.X))
        .attr('cy', d => this.getRenderY(d.Y))
        .attr('class', d => {
        switch (d.AgentType) {
            case 0:
                return 'intersection-node';
                break;
            case 1:
                return 'spawner-node';
                break;
            case 2:
                return 'sink-node';
                break;
            default:
                return 'road-node';
        }
        });
    }

    renderUpdate() {
        const t = this.currentTransform;
        select('#vehicle-layer')
        .selectAll('image')
        .data(this.vehicle_data, d => d.Vehicle_ID)
        .join('image')
        .attr('href','/static/images/bus-logo.png')
        .attr('width', 30/t.k)
        .attr('height', 30/t.k)
        .attr('x', d => this.getRenderX(d.X)-(15/t.k))
        .attr('y', d => this.getRenderY(d.Y)-(15/t.k))
        .attr('opacity', d => d.Status);
    }

    vehicleZoomUpdate() {
        const t = this.currentTransform;
        select('#vehicle-layer')
        .selectAll('image')
        .data(this.vehicle_data, d => d.Vehicle_ID)
        .join('image')
        .attr('x', d => this.getRenderX(d.X)-(15/t.k))
        .attr('y', d => this.getRenderY(d.Y)-(15/t.k))
        .attr('width', 30/t.k)
        .attr('height', 30/t.k);
    }

    getRenderX(world_x) {
    return world_x*this.scaleFactor + this.xScaleOffset;
    }

    getRenderY(world_y) {
    return world_y*-1*this.scaleFactor + this.yScaleOffset;
    }
}