import {Circle, Fill, Line, Stroke, TextElem, Translate, Tspan} from "./svg.js";
import {Bounds, ChartArea} from "./types.js";
import {gold, plum} from "./colors.js";
import {DataSet} from "./stats";

export function addCursor(svgElem: SVGElement, dataSet: DataSet,
                          dimensions: ChartArea, bounds: Bounds) {
    const cursor = Fill("none", Stroke("#C0C0C0", Line(
        0, dimensions.paddingTop,
        0, dimensions.height - dimensions.paddingBottom,
    )))
    svgElem.appendChild(cursor)

    const cursorText = TextElem("0", dimensions.paddingTop.toString(), "")
    cursorText.setAttribute("dx", "1em")
    cursorText.setAttribute("dy", "1em")
    svgElem.appendChild(cursorText)

    const fiveDot = Fill(gold, Stroke("none", Circle(0, 0, 4)))
    svgElem.appendChild(fiveDot)

    const thirtyDot = Fill(plum, Stroke("none", Circle(0, 0, 4)))
    svgElem.appendChild(thirtyDot)

    svgElem.addEventListener("mousemove", updateCursor(dataSet, cursor,
        cursorText, fiveDot, thirtyDot, dimensions, bounds))
}

export function updateCursor(dataSet: DataSet, cursor: SVGElement,
                             cursorText: SVGTextElement, fiveDot: SVGCircleElement,
                             thirtyDot: SVGCircleElement, dimensions: ChartArea,
                             bounds: Bounds
): (evt: MouseEvent) => void {
    return ((evt: MouseEvent) => {
        const div = document.getElementById("chart_container")

        if (div === undefined || div === null) {
            throw new Error("could not find div with ID chart_container")
            return
        }

        const x = Math.max(
            Math.min(
                evt.pageX - div.offsetLeft,
                dimensions.width - dimensions.paddingRight
            ),
            dimensions.paddingLeft,
        )

        const y = Math.max(
            Math.min(
                evt.pageY - div.offsetTop,
                dimensions.height - dimensions.paddingBottom,
            ),
            dimensions.paddingTop,
        )

        cursor.setAttribute("x1", x.toString())
        cursor.setAttribute("x2", x.toString())

        // x ranges from paddingLeft to (width-paddingRight)
        const frac = (x - dimensions.paddingLeft) /
            (dimensions.width - dimensions.paddingLeft - dimensions.paddingRight)
        const cursorDate = new Date(
            bounds.minX +
            (bounds.maxX - bounds.minX) * frac
        )

        const fiveDay = dataSet.ValueAt(dp => dp.fiveDay, cursorDate)
        fiveDot.setAttribute("cx", x.toString())
        fiveDot.setAttribute("cy", dimensions.ScaleY(fiveDay, bounds).toString())

        const thirtyDay = dataSet.ValueAt(dp => dp.thirtyDay, cursorDate)
        thirtyDot.setAttribute("cx", x.toString())
        thirtyDot.setAttribute("cy", dimensions.ScaleY(thirtyDay, bounds).toString())

        const text = cursorText
        Translate(text, x, y)

        if (x / dimensions.width > 0.75) {
            text.setAttribute("text-anchor", "end")
            text.setAttribute("dx", "-.1em")
        }
        if (x / dimensions.width < 0.25) {
            text.setAttribute("text-anchor", "beginning")
            text.setAttribute("dx", "1em")
        }

        while (cursorText.firstChild != null) {
            cursorText.removeChild(cursorText.firstChild)
        }

        text.appendChild(Tspan(`${cursorDate.toDateString()}`))

        const fivespan = Tspan(`5-Day: ${fiveDay.toFixed(1)}`)
        fivespan.setAttribute("x", "0")
        fivespan.setAttribute("dx", text.getAttribute("dx") || "")
        fivespan.setAttribute("dy", "1em")
        text.appendChild(fivespan)

        const thirtyspan = Tspan(`30-Day: ${thirtyDay.toFixed(1)}`)
        thirtyspan.setAttribute("x", "0")
        thirtyspan.setAttribute("dx", text.getAttribute("dx") || "")
        thirtyspan.setAttribute("dy", "1em")
        text.appendChild(thirtyspan)
    })
}