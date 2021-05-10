export class DataSet {
    data: Array<DataPoint>

    constructor(data?: Array<DataPoint>) {
        if (data != null) {
            this.data = data
        } else {
            this.data = new Array<DataPoint>()
        }
    }

    Points(retriever: ((dp: DataPoint) => number), includeZero?: boolean): Array<Point> {
        const ret = new Array<Point>()

        this.data.forEach(value => {
            const v = retriever(value)
            if (v == 0 && !includeZero) {
                return
            }
            ret.push({x: value.date.valueOf(), y: v})
        })

        return ret
    }

    ValueAt(retriever: ((dp: DataPoint) => number), x: Date): number {
        let left: DataPoint | null = null
        let right: DataPoint | null = null

        for (const dp of this.data) {
            const v = retriever(dp)
            if (v == 0) continue

            if (dp.date <= x && (left === null || dp.date > left.date)) {
                left = dp
            }

            if (dp.date >= x && (right === null || dp.date < right.date)) {
                right = dp
            }
        }

        if (left === null) {
            return 0
        }
        if (right === null) {
            return 0
        }

        if (left == right) {
            return retriever(left)
        }

        // consider
        // (1,1)  ... (5,?) ... (10,10)
        // we know 2/2, 5/5, etc.

        const rv = retriever(right)
        const lv = retriever(left)
        const dy = rv - lv
        const dx = right.date.valueOf() - left.date.valueOf()

        return lv + (x.valueOf() - left.date.valueOf()) / dx * dy
    }
}

export interface Point {
    x: number
    y: number
}

export interface DataPoint {
    date: Date
    day: number
    fiveDay: number
    thirtyDay: number
}