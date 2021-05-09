import {Circle, Fill, Line, Path, Stroke, TextElem, TitleElem} from './svg.js'
import {DataPoint, DataSet} from "./stats.js";
import {Bounds, ChartArea} from "./types.js";
import {addCursor} from "./cursor.js";
import {gold, plum} from "./colors.js";

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

    fetch(path)
        .then(response => {
            if (!response.ok) {
                throw new Error(`error fetching data; status: ${response.status}`);
            }
            return response.blob()
        })
        .then(blob => blob.text())
        .then(data => {
            const ds = new DataSet(JSON.parse(data).map(decodeWeight(useKg)))
            if (ds.data.length == 0) {
                noData(svgElem)
            } else {
                applyData(svgElem, dimensions, ds)
            }
        })
}

interface Weight {
    Date: string
    Day: number
    FiveDay: number
    ThirtyDay: number
}

function decodeWeight(kg: boolean): ((w: Weight) => DataPoint) {
    return w => {
        const mul = kg ? 1 : 1 / 0.45359237
        const dp: DataPoint = {
            date: new Date(w.Date),
            day: w.Day * mul,
            fiveDay: w.FiveDay * mul,
            thirtyDay: w.ThirtyDay * mul,
        }
        return dp
    }
}

function noData(svgElem: SVGElement) {
    const text = TextElem("50%", "50%", "No data in range.")
    svgElem.appendChild(text)
}

function applyData(svgElem: SVGElement, dimensions: ChartArea, data: DataSet) {
    const bounds = dataBounds(data);
    drawGridLines(svgElem, dimensions, bounds)

    data.data.forEach(value => {
        if (value.day == 0) {
            return
        }
        svgElem.appendChild(
            Fill("green",
                TitleElem(value.date.toDateString(),
                    Circle(
                        dimensions.ScaleX(value.date.valueOf(), bounds),
                        dimensions.ScaleY(value.day, bounds),
                        2
                    ))))
    })

    const fivePath = Fill("none", Stroke(gold,
        Path(data.Points(dp => dp.fiveDay).map(value => {
            return {
                x: dimensions.ScaleX(value.x, bounds),
                y: dimensions.ScaleY(value.y, bounds)
            }
        }))))
    svgElem.appendChild(fivePath)

    const thirtyPath = Fill("none", Stroke(plum,
        Path(data.Points(dp => dp.thirtyDay).map(value => {
            return {
                x: dimensions.ScaleX(value.x, bounds),
                y: dimensions.ScaleY(value.y, bounds)
            }
        }))))
    svgElem.appendChild(thirtyPath)

    addCursor(svgElem, data, dimensions, bounds)
}

function dataBounds(data: DataSet): Bounds {
    let ret: Bounds = {
        maxX: Number.MIN_VALUE,
        maxY: Number.MIN_VALUE,
        minX: Number.MAX_VALUE,
        minY: Number.MAX_VALUE
    }

    data.data.forEach((dp: DataPoint) => {
        if (dp.day == 0 && dp.fiveDay == 0 && dp.thirtyDay == 0) {
            return
        }

        if (dp.date.valueOf() < ret.minX) {
            ret.minX = dp.date.valueOf()
        }
        if (dp.date.valueOf() > ret.maxX) {
            ret.maxX = dp.date.valueOf()
        }
        for (const v of [dp.day, dp.fiveDay, dp.thirtyDay]) {
            if (v == 0) {
                continue
            }
            ret.minY = Math.min(ret.minY, v)
            ret.maxY = Math.max(ret.maxY, v)
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
        const text = TextElem("0", dimensions.ScaleY(y, bounds).toString(), y.toFixed(1))
        svg.appendChild(text)
        svg.appendChild(Fill("none", Stroke("#C0C0C0", Line(
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
        const label = TextElem(sx.toString(), sy.toString(), text)
        label.setAttribute("transform",
            `rotate(-90, ${sx.toString()}, ${sy.toString()})`)
        label.setAttribute("textLength", dimensions.paddingBottom.toString())
        svg.appendChild(label)

        svg.appendChild(Fill('none', Stroke('#F0F0F0', Line(
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

for (const elem of document.getElementsByClassName("timeclick")) {
    elem.addEventListener("click", () => {
        updateChart(parseInt(elem.getAttribute("data-days") || "0"))
        return false
    })
}