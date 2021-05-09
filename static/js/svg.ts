export function TextElem(x: string, y:string, text: string): SVGTextElement {
    const elem = document.createElementNS("http://www.w3.org/2000/svg", "text")
    elem.setAttribute("x", x)
    elem.setAttribute("y", y)
    elem.innerHTML = text
    return elem
}

export function Fill<U extends SVGElement>(color: string, e: U): U {
    e.setAttribute("fill", color)
    return e
}

export function Stroke<U extends SVGElement>(color: string, e: U): U {
    e.setAttribute("stroke", color)
    return e
}

export function Line(x1: number, y1: number, x2: number, y2: number): SVGElement {
    const line = document.createElementNS("http://www.w3.org/2000/svg", "line")
    line.setAttribute("x1", x1.toString())
    line.setAttribute("x2", x2.toString())
    line.setAttribute("y1", y1.toString())
    line.setAttribute("y2", y2.toString())
    return line
}

export function Circle(cx: number, cy: number, r: number): SVGCircleElement {
    const circle = document.createElementNS(
        "http://www.w3.org/2000/svg",
        "circle")
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

export function Path(points: Array<Point>): SVGPathElement {
    const pathElem = document.createElementNS(
        "http://www.w3.org/2000/svg",
        "path") as SVGPathElement

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

export function Tspan(text: string): SVGElement {
    const tspan = document.createElementNS("http://www.w3.org/2000/svg", "tspan") as SVGTSpanElement
    tspan.textContent = text
    return tspan
}

export function Translate<U extends SVGElement>(e: U, x: number, y: number): U {
    e.setAttribute("transform", `translate(${x} ${y})`)
    return e
}