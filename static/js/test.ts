import {DataPoint, DataSet} from "./stats";

function Fixture(): DataSet {
    const days = [
        new Date(1, 2, 3, 4),
        new Date(1, 2, 3, 5),
        new Date(1, 2, 4),
        new Date(1, 2, 5),
        new Date(1, 2, 6),
        new Date(1, 2, 7),
    ]

    return new DataSet(
        days.map<DataPoint>((d: Date, i: number) => {
            return {
                date: d, value: i,
            }
        }),
    )
}

function TestByDay() {
    const byday = Fixture().ByDay()

    if (byday.size != 5) {
        for (const entry of byday.entries()) {
            console.log(entry)
        }
        throw new Error(`by-day should collapse first two data points, but returned ${byday.size}`)
    }
}

function TestMovingAverage() {
    const avg = Fixture().MovingAverage(5)
    if (avg.data.length != 3) {
        for (const entry of avg.data.entries()) {
            console.log(entry)
        }
        throw new Error(`MovingAverage(5) should have three data points, but ${avg.data.length}`)
    }
}

TestByDay()
TestMovingAverage()