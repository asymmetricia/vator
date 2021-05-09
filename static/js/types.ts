export class ChartArea {
    width: number
    height: number
    paddingTop: number
    paddingBottom: number
    paddingLeft: number
    paddingRight: number

    constructor(
        width: number,
        height: number,
        paddingTop: number,
        paddingBottom: number,
        paddingLeft: number,
        paddingRight: number,
    ) {
        this.width = width
        this.height = height
        this.paddingTop = paddingTop
        this.paddingBottom = paddingBottom
        this.paddingLeft = paddingLeft
        this.paddingRight = paddingRight
    }

    ScaleY(y: number, bounds: Bounds): number {
        const frac = (y - bounds.minY) / (bounds.maxY - bounds.minY)
        const scaled = (1 - frac) * (this.height - this.paddingTop - this.paddingBottom)
        return this.paddingTop + scaled
    }

    ScaleX(x: number, bounds: Bounds): number {
        const frac = (x - bounds.minX) / (bounds.maxX - bounds.minX)
        const scaled = frac * (this.width - this.paddingLeft - this.paddingRight)
        return this.paddingLeft + scaled
    }
}

export interface Bounds {
    minX: number;
    maxX: number;
    minY: number;
    maxY: number;
}
