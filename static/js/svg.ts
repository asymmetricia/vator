export function TextElem(x: number, y: number, text: string): SVGElement {
    const elem = document.createElementNS("http://www.w3.org/2000/svg", "text")
    elem.setAttribute("x", x.toString())
    elem.setAttribute("y", y.toString())
    elem.innerHTML = text
    return elem
}

export function FillElem(color: string, e: SVGElement): SVGElement {
    e.setAttribute("fill", color)
    return e
}

export function StrokeElem(color: string, e: SVGElement): SVGElement {
    e.setAttribute("stroke", color)
    return e
}

export function LineElem(x1: number, y1: number, x2: number, y2: number): SVGElement {
    const line = document.createElementNS("http://www.w3.org/2000/svg", "line")
    line.setAttribute("x1", x1.toString())
    line.setAttribute("x2", x2.toString())
    line.setAttribute("y1", y1.toString())
    line.setAttribute("y2", y2.toString())
    return line
}

export function CircleElem(cx: number, cy: number, r: number): SVGElement {
    const circle = document.createElementNS("http://www.w3.org/2000/svg", "circle")
    circle.setAttribute("cx", cx.toString())
    circle.setAttribute("cy", cy.toString())
    circle.setAttribute("r", r.toString())
    return circle
}

export function TitleElem(title: string, e: SVGElement): SVGElement {
    const titleElem = document.createElementNS("http://www.w3.org/2000/svg", "title")
    titleElem.innerHTML = title
    e.appendChild(titleElem)
    return e
}

interface Point {
    x: number
    y: number
}

export function PathElem(points: Array<Point>): SVGElement {
    const pathElem = document.createElementNS("http://www.w3.org/2000/svg", "path")

    let cmd = ""
    points.forEach(pt => {
        if (cmd == "") {
            cmd = "M "
        } else {
            cmd = cmd + "L "
        }

        cmd = cmd + `${pt.x} ${pt.y} `
    })

    pathElem.setAttribute("d", cmd)
    pathElem.setAttribute("style", "stroke-width: 1px; stroke-linejoin: round;")

    return pathElem
}