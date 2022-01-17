import ReactDOM from "react-dom";
import React, { useState, useEffect } from "react";
import { Flipper, Flipped } from "react-flip-toolkit";
import shuffle from "lodash.shuffle";
import "./styles.css";

function compareDecimals(a, b) {
  if (a.count === b.count) return 0;

  return a.count < b.count ? -1 : 1;
}

const ListShuffler = () => {
  const [data, setData] = useState([])
  const sortList = () => {
    let copy = data.slice();
    copy.sort(compareDecimals);
    setData(copy);
  };

  const lst2str = (data) => {
    let out = "";
    for (let i = 0; i < data.length; i++) {
      out += data[i].count.toString() + data[i].word.toString();
    }

    return out;
  }

  const updateData = () => {
    fetch('/api/words/top')
      .then(response => response.json())
      .then(data => setData(data));
  }

  useEffect(() => {
    // TODO: Make this into a web socket
    setInterval(updateData, 1000);
  }, [])

  return (
    <div id="shuffle">
      <Flipper flipKey={lst2str(data)}>
        <table>
          {data.map(({count, word}) => (
            <Flipped key={word} flipId={word}>
              <tr className="list-item card">
                <td>{count}</td>
                <td>{word}</td>
              </tr>
            </Flipped>
          ))}
        </table>
      </Flipper>
    </div>
  );
};

ReactDOM.render(<ListShuffler />, document.querySelector("#root"));
