import React, { useState, useEffect } from "react";
import Button from '@mui/material/Button';
import Grid from "@mui/material/Grid";
import ArrowForwardRoundedIcon from '@mui/icons-material/ArrowForwardRounded';
import { Base64 } from 'js-base64';
import { Flipped } from "react-flip-toolkit";


// react-flip-tooklkit does not support sub components having their own state.
// because of this, every WordRow must be immutable after construction...

export const WordRow = (props) => {
  let letterGrade = "A+"

  const getLetterGrade = (wordScore) => {
    let grades =      ["A+", "A", "A-", "B+", "B", "B-", "C", "D", "D-"];
    let breakPoints = [0.9,  0.7,  0.6,  0.5,  0.4, 0.3, 0.2, 0.1, 0.0];
    for (let i = 0; i < breakPoints.length; i++) {
        if (wordScore >= breakPoints[i]) {
            return grades[i];
        }
    }

    return grades[grades.length - 1];
  }

  const getEmoji = (wordScore) => {
      if (wordScore >= 0.7) {
          return "🔥";
      }
      return ""
  }

  const hasEmoji = (wordScore) => {
      return getEmoji(wordScore) == "";
  }

  const round = (n, digits) => {
      return Math.floor((n * Math.pow(10, digits))) / Math.pow(10, digits)
  }

  const translate = () => {
    try {
        // the Base64 library has an encodeURI but it removes the padding for you with no option to keep it,
        // so I snipped this part of the source code and pasted it here, while keeping the padding
        let urlSafe = Base64.encode(props.word).replace(/[+\/]/g, function (m0) { return m0 == '+' ? '-' : '_'; });
        fetch('/api/translate?word=' + urlSafe)
        .then(response => response.json())
        .then(data => {
            // NOTE: This improves latency since this would otherwise be updated a max of 1 second later.
            // this is really bad for coupling. I have to do it because of react-flip-toolkit
            props.updateTranslation(props.word, data["translation"]);
        });
    } catch(err) {
        console.log(err)
    }
  }
  return (
    <Flipped key={props.word} flipId={props.word}>
        <li className="list-item card">
            <Grid
                container
                justifyContent="space-between"
                alignItems="center"
                spacing={2}
            >
            <Grid item md={5} xs={12} overflow="hidden">
                <Grid container spacing={0}>
                    <Grid item md={12} xs={12}>
                        <p
                            style={{
                            margin: 0,
                            textOverflow: "ellipsis",
                            overflow: "hidden",
                            width: "100%",
                            display: "inline",
                            marginRight: "5px"
                            }}
                        >
                            {props.word}
                        </p>
                    </Grid>
                    <Grid item md={12} xs={12} display="inline-flex" alignItems="flex-end">
                        <p className="translation" style={{margin: 0, marginTop: "3px"}}>
                            {/* I blame the cloud translation api. I need to fix this in the future. */}
                            {
                                props.translation && <span dangerouslySetInnerHTML={{ __html: "(" + props.translation + ")" }} />
                            }
                        </p>
                    </Grid>
                </Grid>
            </Grid>
            <Grid item md={7} xs={12}>
                <Grid container alignItems="center" justifyContent="flex-end" textAlign="right" spacing={4}>
                    <Grid item md={3}>
                        {(!props.translation) && <Button
                        variant="outlined"
                        size="small"
                        onClick={translate}
                        style={{marginTop: "-5px"}}
                        >
                        Translate
                        </Button>}
                    </Grid>
                    { hasEmoji(props.wordScore && 
                        <Grid item md={1}>
                            {getEmoji(props.wordScore)}
                        </Grid>                
                    )}
                    <Grid item md={4} textAlign="left">
                        <p className="stats-p">{getLetterGrade(props.wordScore)}</p>
                        <p style={{fontSize: "1rem", fontWeight: 400, margin: 0}}>({`${round(props.wordScore, 2)} ${round(props.multiple, 1)}x ${props.count}`})</p>
                    </Grid>
                    <Grid item md={3}>
                        <Grid container justifyContent="center" alignItems="center">
                        <Button
                        variant="contained"
                        size="large"
                        onClick={() => window.open('https://twitter.com/search?q=' + encodeURIComponent(props.word), '_')}
                        >
                            <ArrowForwardRoundedIcon />
                        </Button>
                        </Grid>
                    </Grid>
                </Grid>
            </Grid>
        </Grid>
        </li>
    </Flipped>
  )
}