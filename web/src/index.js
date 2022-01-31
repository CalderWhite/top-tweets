import ReactDOM from "react-dom";
import React, { useState, useEffect } from "react";
import { Flipper } from "react-flip-toolkit";
// import shuffle from "lodash.shuffle";

import "./styles.css";
import { WordRow } from "./WordRow";
import Button from '@mui/material/Button';
import Grid from "@mui/material/Grid";
import ArrowForwardRoundedIcon from '@mui/icons-material/ArrowForwardRounded';


const App = () => {
  const [data, setData] = useState([])
  const [totalTweets, setTweets] = useState(0);

  const lst2str = (data) => {
    let out = "";
    for (let i = 0; i < data.length; i++) {
      out += data[i].count.toString() + data[i].word.toString();
    }

    return out;
  }

  const updateData = () => {
    try {
      fetch('/api/words/top')
      .then(response => response.json())
      .then(data => {
        setTweets(data["total"])
        setData(data["words"])
      });
    } catch(err) {
      console.log(err)
    }
  }

  // this is a terrible way of updating translations and is 100% coupled with WordRow.
  // However, react-flip-toolkit has forced me to do it this way.
  const updateTranslation = (word, translation) => {
    let data2 = data.slice()
    for (let i = 0; i < data2.length; i++) {
      if (data2[i].word == word) {
        data2[i].translation = translation;
      }
    }

    setData(data2);
  }

  useEffect(() => {
    // TODO: Make this into a web socket
    updateData();
    setInterval(updateData, 1000);
  }, [])

  return (
    <div>
      <div className="table-wrapper">
        <Grid container justifyContent="flex-end" >
          <Grid item md={3}>
            <p
            style={{
              margin: "15px",
              textAlign: "left",
              fontFamily: "Consolas, monaco, monospace",
              fontSize: "14px",
              background: "white",
            }}
          >
            Total Tweets: {totalTweets.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",")}
          </p>
          </Grid>
        </Grid>
      </div>
      <div id="shuffle">
        <Flipper flipKey={lst2str(data)}>
          <ul className="table-wrapper">
            {data.map(({wordScore, multiple, count, word, translation}) => (
              <WordRow
                word={word}
                translation={translation}
                count={count}
                multiple={multiple}
                wordScore={wordScore}
                updateTranslation={updateTranslation}
                />
            ))}
          </ul>
        </Flipper>
      </div>
    </div>
  );
};

ReactDOM.render(<App />, document.querySelector("#root"));
