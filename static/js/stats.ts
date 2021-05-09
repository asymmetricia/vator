export class DataSet {
    data: Array<DataPoint>

    constructor(data?: Array<DataPoint>) {
        if (data != null) {
            this.data = data
        } else {
            this.data = new Array<DataPoint>()
        }
    }

    ByDay(): Map<Date, number> {
        let ret = new Map<Date, number>()
        let points = new Map<string, Array<number>>()
        this.data.forEach(point => {
            let d = new Date(
                point.date.getUTCFullYear(),
                point.date.getUTCMonth(),
                point.date.getUTCDay()
            ).toJSON()
            let pt = points.get(d)
            if (!pt) {
                pt = new Array<number>()
            }
            points.set(d, pt.concat(point.value))
        })

        points.forEach((vs, d) => {
            let sum = 0
            vs.forEach(v => sum += v)
            ret.set(new Date(d), sum / vs.length)
        })

        return ret
    }

    MovingAverage(days: number, xFactor?: number): DataSet {
        const ret = new DataSet()
        if (xFactor == null) {
            xFactor = Math.floor(days * 3 / 4)
        }

        const data = new Map<string, number>()
        this.ByDay().forEach((value, key) =>
            data.set(key.toJSON(), value)
        )
        const keys = Array.from(data.keys())
        const begin: Date = new Date(keys[0])
        const end: Date = new Date(keys[keys.length - 1])
        for (let date = new Date(begin); date <= end; date.setDate(date.getDate() + 1)) {
            let sum: number = 0, count: number = 0
            for (let j = 0; j < days; j++) {
                const cursor = new Date(date)
                cursor.setDate(cursor.getDate() - j)
                const d = data.get(cursor.toJSON())
                if (d !== undefined) {
                    sum += d
                    count++
                }
            }
            if (count >= xFactor) {
                ret.data.push({date: date, value: sum / count})
            }
        }

        return ret
    }

    Points(): Array<Point> {
        const ret = new Array<Point>()

        this.data.forEach(value => {
            ret.push({x: value.date.valueOf(), y: value.value})
        })

        return ret
    }
}

interface Point {
    x: number
    y: number
}

export interface DataPoint {
    date: Date
    value: number
}