import {CircleElem, FillElem, LineElem, PathElem, StrokeElem, TextElem, TitleElem} from './svg.js'
import {DataPoint, DataSet} from "./stats.js";

class ChartArea {
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

function updateChart(days?: number) {
    const container = document.getElementById("chart_container");
    if (!container) {
        throw new Error(`element with id chart_container not found`);
    }

    const dimensions: ChartArea = new ChartArea(
        container.offsetWidth,
        container.offsetHeight - 4,
        20,
        80,
        40,
        40)

    const svgElemList = document.getElementsByTagNameNS("http://www.w3.org/2000/svg", "svg")

    let svgElem: SVGElement
    if (svgElemList.length > 0) {
        svgElem = svgElemList[0]
    } else {
        svgElem = document.createElementNS("http://www.w3.org/2000/svg", "svg")
        svgElem.setAttribute("name", "svg")
        container.appendChild(svgElem)
    }

    while (svgElem.firstChild != null) {
        svgElem.removeChild(svgElem.firstChild)
    }

    svgElem.setAttribute(
        "viewBox",
        "0 0 " + dimensions["width"].toString() + " " + dimensions["height"].toString()
    )


    const params = new URLSearchParams(window.location.search)
    const user = params.get('user')

    if (days != null) {
        params.set("days", days.toString())
        history.pushState(null, document.title, "/graph?" + params.toString())
    }

    const dayStr = params.get('days')
    days = dayStr ? parseInt(dayStr) : 365

    const path = '/data?' +
        'days=' + days.toString() +
        (user == null ? '' : '&user=' + encodeURIComponent(user))

    const useKg = new URLSearchParams(window.location.search).get('kg') == "true";

    let data = fetch(path).then(response => {
        if (!response.ok) {
            throw new Error(`error fetching data; status: ${response.status}`);
        }
        return response.blob()
    })
        .then(blob => {
            return blob.text()
        })
        .then(data => {
            const ds = new DataSet(JSON.parse(data).map(decodeWeight(useKg)))
            applyData(svgElem, dimensions, ds)
        })
}

interface Weight {
    Date: string
    Kgs: number
}

function decodeWeight(kg: boolean): ((w: Weight) => DataPoint) {
    return w => {
        const dp = {
            date: new Date(Date.parse(w.Date)),
            value: w.Kgs
        }
        if (!kg) {
            dp.value /= 0.45359237
        }
        return dp
    }
}

function applyData(svgElem: SVGElement, dimensions: ChartArea, data: DataSet) {
    const bounds = dataBounds(data);
    drawGridLines(svgElem, dimensions, bounds)

    data.data.forEach(value => {
        svgElem.appendChild(
            FillElem("green",
                TitleElem(value.date.toDateString(),
                    CircleElem(
                        dimensions.ScaleX(value.date.valueOf(), bounds),
                        dimensions.ScaleY(value.value, bounds),
                        2
                    ))))
    })

    svgElem.appendChild(PathElem(data.MovingAverage(5).Points().map(value => {
        console.log(value)
        return {
            x: dimensions.ScaleX(value.x, bounds),
            y: dimensions.ScaleY(value.y, bounds)
        }
    })))
}

interface Bounds {
    minX: number;
    maxX: number;
    minY: number;
    maxY: number;
}

function dataBounds(data: DataSet): Bounds {
    let ret: Bounds = {
        maxX: Number.MIN_VALUE,
        maxY: Number.MIN_VALUE,
        minX: Number.MAX_VALUE,
        minY: Number.MAX_VALUE
    }

    data.data.forEach((value: DataPoint) => {
        if (value.date.valueOf() < ret.minX) {
            ret.minX = value.date.valueOf()
        }
        if (value.date.valueOf() > ret.maxX) {
            ret.maxX = value.date.valueOf()
        }
        if (value.value < ret.minY) {
            ret.minY = value.value
        }
        if (value.value > ret.maxY) {
            ret.maxY = value.value
        }
    })

    return ret
}

const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

function formatDate(d: Date, day: boolean): string {
    let ret: string = months[d.getMonth()] + ' ' + d.getFullYear().toString()
    if (day) {
        ret = d.getDate().toString() + ' ' + ret
    }
    return ret
}

function drawGridLines(svg: SVGElement, dimensions: ChartArea, bounds: Bounds) {
    for (let yIdx: number = 0; yIdx < 12; yIdx++) {
        const y = bounds.minY + (bounds.maxY - bounds.minY) * yIdx / 12
        const text = TextElem(0, dimensions.ScaleY(y, bounds), y.toFixed(1))
        svg.appendChild(text)
        svg.appendChild(FillElem("none", StrokeElem("#C0C0C0", LineElem(
            dimensions.ScaleX(bounds.minX, bounds),
            dimensions.ScaleY(y, bounds),
            dimensions.ScaleX(bounds.maxX, bounds),
            dimensions.ScaleY(y, bounds),
        ))))
    }

    const begin = new Date(bounds.minX)
    const end = new Date(bounds.maxX)
    const days = (end.valueOf() - begin.valueOf()) / 1000 / 86400

    if (days > 90) {
        end.setDate(1)
    }

    for (let x = end; x >= begin;) {
        const sx = dimensions.ScaleX(x.valueOf(), bounds)
        const sy = dimensions.height

        const text = formatDate(x, days <= 90)
        const label = TextElem(sx, sy, text)
        label.setAttribute("transform",
            `rotate(-90, ${sx.toString()}, ${sy.toString()})`)
        label.setAttribute("textLength", dimensions.paddingBottom.toString())
        svg.appendChild(label)

        svg.appendChild(FillElem('none', StrokeElem('#F0F0F0', LineElem(
            sx, dimensions.ScaleY(bounds.minY, bounds),
            sx, dimensions.ScaleY(bounds.maxY, bounds),
        ))))

        if (days > 365) {
            x.setMonth(x.getMonth() - 3)
        } else if (days > 90) {
            x.setMonth(x.getMonth() - 1)
        } else if (days > 30) {
            x.setDate(x.getDate() - 7)
        } else {
            x.setDate(x.getDate() - 1)
        }
    }
}

document.addEventListener("DOMContentLoaded", () => updateChart())
window.onresize = () => updateChart()